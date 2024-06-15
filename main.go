package main

/*
This is an application that runs tSQLt tests against a database and prints the
results in an easy to read TUI (courtesy of Bubble Tea).

It reads a list of tests from stdin in the following format:
	TestSuite.TestName
	-- or just the suite may be specified
	TestSuite

Database connection details are read from environment variables, or may be
specified in the command-line options:
	-s server   -- or $TSQLR_SERVER
	-d database -- or $TSQLR_DATABASE
	-u user     -- or $TSQLR_USER
	-p password -- or $TSQLR_PASSWORD
*/

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tsqlr/dbutil"
	"tsqlr/table"
	t "tsqlr/tests"

	// tea "github.com/charmbracelet/bubbletea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/denisenkom/go-mssqldb"
)

type dbConfig struct {
	server   string
	database string
	user     string
	password string
}

func (conf dbConfig) open() (*sql.DB, *dbutil.Logger) {
	logger := dbutil.Logger{Results: map[string][]string{}}
	mssql.SetContextLogger(logger)
	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(conf.user, conf.password),
		Host:     conf.server,
		RawQuery: url.Values{"database": {conf.database}, "log": {"2"}}.Encode(),
	}

	connector, err := mssql.NewConnector(u.String())
	if err != nil {
		log.Fatalln(err.Error())
	}

	connector.SessionInitSQL = "SET NOCOUNT ON;"

	var db *sql.DB = sql.OpenDB(connector)

	fmt.Printf("Connecting to server %s...\r", conf.server)
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalln(err.Error())
	}

	return db, &logger
}

type cmdOpts struct {
	db       dbConfig
	testfile *string
}

func parseOpts() cmdOpts {
	var server, database, user, password, testfile string

	flag.StringVar(&server, "s", "", "Database server (default: $TSQLR_SERVER)")
	flag.StringVar(&database, "d", "", "Database name (default: $TSQLR_DATABASE)")
	flag.StringVar(&user, "u", "", "Database username (default: $TSQLR_USER)")
	flag.StringVar(&password, "p", "", "Database user password (default: $TSQLR_PASSWORD)")

	flag.StringVar(&testfile, "f", "", "Test file (stdin if not specified)")

	flag.Parse()

	if server == "" {
		if server = os.Getenv("TSQLR_SERVER"); server == "" {
			log.Fatalln("missing -s server")
		}
	}
	if database == "" {
		if database = os.Getenv("TSQLR_DATABASE"); database == "" {
			log.Fatalln("missing -d database")
		}
	}
	if user == "" {
		if user = os.Getenv("TSQLR_USER"); user == "" {
			log.Fatalln("missing -u user")
		}
	}
	if password == "" {
		if password = os.Getenv("TSQLR_PASSWORD"); password == "" {
			log.Fatalln("missing -p password")
		}
	}

	var _testfile *string
	if testfile == "" {
		_testfile = nil
	} else if _, err := os.Stat(testfile); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("test file not found: %s\n", testfile)
	} else {
		_testfile = &testfile
	}

	db := dbConfig{server, database, user, password}
	return cmdOpts{db, _testfile}
}

func main() {
	opts := parseOpts()
	tests := parseTestFile(opts.testfile)

	// TODO: implement timeout (5s?)
	conn, logger := opts.db.open()
	defer conn.Close()

	queue := make(chan *t.Test)
	p := tea.NewProgram(table.InitialModel(queue, tests))

	go processTestQueue(conn, logger, queue, p)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() { // handle signals
		_ = <-sigs
		conn.Close()
		p.Send(tea.Quit())
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		conn.Close()
		os.Exit(1)
	}
}

func parseTestFile(testfile *string) []t.Test {
	var scanner *bufio.Scanner
	if testfile == nil {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		file, err := os.Open(*testfile)
		if err != nil {
			log.Fatalln(err.Error())
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}
	tests := []t.Test{}
	for scanner.Scan() {
		line := scanner.Text()
		// trim whitespace
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var bom rune = '\uFEFF'
		if strings.ContainsRune(line, bom) {
			line = strings.TrimLeft(line, "\uFEFF")
		}

		pieces := strings.Split(line, ".")
		var suite, name string
		if len(pieces) == 1 {
			suite = pieces[0]
		} else if len(pieces) == 2 {
			suite = pieces[0]
			name = pieces[1]
		} else {
			log.Fatalf("invalid test line: %s\n", line)
		}

		tests = append(tests, t.Test{Suite: suite, Name: name})
	}

	if err := scanner.Err(); err != nil {
		log.Fatalln(err.Error())
	}

	if len(tests) == 0 {
		log.Fatalln("no tests found")
	}

	return tests
}

func runTest(db *sql.DB, logger *dbutil.Logger, test *t.Test) (results []string, err error) {
	var ctx context.Context
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	ctx = context.WithValue(ctx, "testname", test.String())
	defer cancel()

	logger.ClearResults(test) // clear old results in case of rerun
	_, err = db.ExecContext(ctx,
		"EXEC tSQLt.Run @test",
		sql.Named("test", test.String()))

	var ok bool
	results, ok = logger.GetResults(test)
	if !ok {
		err = fmt.Errorf("No results for test: %s", test)
	}

	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "Test Case Summary") {
			if strings.HasPrefix(msg, "mssql: ") {
				_, msg, _ = strings.Cut(msg, "mssql: ")
			}
			results = append(results, msg)
			err = nil
		}
	}

	return
}

func processTestQueue(conn *sql.DB, logger *dbutil.Logger, queue chan *t.Test, p *tea.Program) {
	for {
		test := <-queue

		var err error
		test.Results, err = runTest(conn, logger, test)

		if err != nil {
			test.Status = t.ERROR
			p.Send("TestUpdated")
			continue
		}

		test.Status, err = test.ProcessResults()
		if err != nil {
			test.Status = t.ERROR
			test.Results = append([]string{err.Error()}, test.Results...)
		}
		p.Send("TestUpdated")
	}
}
