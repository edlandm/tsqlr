package dbutil

import (
	"context"
	"strings"

	t "tsqlr/tests"

	"github.com/denisenkom/go-mssqldb/msdsn"
)

type Logger struct {
	Results map[string][]string
}

func (l Logger) Log(ctx context.Context, category msdsn.Log, msg string) {
	if ctx == nil {
		return
	}

	msg = strings.Trim(msg, " \t\r\n-")
	if msg == "" || msg[0] == '+' {
		return
	}

	value := ctx.Value("testname")
	if value == nil {
		return
	}

	testname := value.(string)
	testResults := l.Results[testname]
	if testResults == nil {
		l.Results[testname] = []string{msg}
		return
	}
	l.Results[testname] = append(testResults, msg)
}

func (l Logger) GetResults(test *t.Test) (results []string, ok bool) {
	testname := test.String()
	results, ok = l.Results[testname]
	return
}

func (l Logger) ClearResults(test *t.Test) (results []string, ok bool) {
	testname := test.String()
	_, ok = l.Results[testname]
	if ok {
		l.Results[testname] = nil
	}
	return
}
