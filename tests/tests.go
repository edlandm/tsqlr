package tests

import (
	"fmt"
	"regexp"
	"strings"
)

type Status int

const (
	INITIAL Status = iota
	RUNNING
	ERROR
	PASS
	FAIL
	MISSING
	UNKNOWN
)

func (s Status) String() string {
	switch s {
	case INITIAL:
		return "PENDING"
	case RUNNING:
		return "RUNNING"
	case ERROR:
		return "ERROR"
	case PASS:
		return "PASS"
	case FAIL:
		return "FAIL"
	case MISSING:
		return "MISSING"
	}
	return "Unknown"
}

type Test struct {
	Suite   string
	Name    string
	Status  Status
	Results []string
}

func (t Test) String() string {
	if t.Name == "" {
		return t.Suite
	}
	return fmt.Sprintf("%s.%s", t.Suite, t.Name)
}

func (t *Test) ProcessResults() (Status, error) {
	isSuite := t.Name == ""
	// I'm leaving these as two different methods for now in case I decide to
	// do  special processing for Suites vs Tests, however they are currently
	// nearly identical and I may choose to merge them in the future.
	if isSuite {
		return t.processSuiteResults()
	}
	return t.processTestResults()
}

func (t *Test) processSuiteResults() (Status, error) {
	if len(t.Results) == 0 {
		return ERROR, fmt.Errorf("no results for suite: %s", t)
	}

	// collect all of the lines until we find summaryLine
	summaryLine := "|Test Execution Summary|"
	var errorResults []string
	for _, line := range t.Results {
		if strings.Contains(line, summaryLine) {
			break
		}
		errorResults = append(errorResults, line)
	}

	if len(errorResults) > 0 {
		t.Results = errorResults
		return FAIL, nil
	}

	return PASS, nil
}

func (t *Test) processTestResults() (Status, error) {
	if len(t.Results) == 0 {
		return ERROR, fmt.Errorf("no results for test: %s", t)
	}

	// collect all of the lines before the start of the summary section
	// these are likely error messages, but could also be anthing printed to
	// stdout
	var outputLines []string
	for _, line := range t.Results {
		if strings.Contains(line, "|Test Execution Summary|") {
			break
		}
		outputLines = append(outputLines, line)
	}

	// find the line containing the final summary. This is usually the last
	// line in the test results, but we want to confirm that instead of
	// assuming
	summaryEnd := "Test Case Summary:"
	var summaryEndLine *string
	for _, line := range t.Results {
		if strings.Contains(line, summaryEnd) {
			summaryEndLine = &line
		}
	}

	if summaryEndLine == nil {
		return ERROR, fmt.Errorf("Failed to find summary line: %s", summaryEnd)
	}

	summaryRegex := regexp.MustCompile(`^Test Case Summary: (\d+) test case\(s\) executed, (\d+) succeeded, (\d+) skipped, (\d+) failed, (\d+) errored\.`)
	matches := summaryRegex.FindStringSubmatch(*summaryEndLine)
	if len(matches) == 0 {
		return ERROR, fmt.Errorf("Failed to parse summary: %s", *summaryEndLine)
	}

	testCases := matches[1]
	succeeded := matches[2]
	skipped := matches[3]
	failed := matches[4]
	errored := matches[5]

	switch {
	case testCases == "0":
		return MISSING, nil
	case testCases == succeeded:
		return PASS, nil
	case testCases == skipped:
		t.Results = outputLines
		return MISSING, nil
	case testCases == failed:
		t.Results = outputLines
		return FAIL, nil
	case testCases == errored:
		t.Results = outputLines
		return ERROR, nil
	default:
		return UNKNOWN, fmt.Errorf("Unknown result: %s", matches[0])
	}
}
