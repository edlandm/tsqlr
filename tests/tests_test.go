package tests

import (
	"testing"
)

func Test_Test_InitialStatus(t *testing.T) {
	mytest := Test{}
	expected := INITIAL
	actual := mytest.Status

	if actual != expected {
		t.Errorf("Expected <%s>, got <%s>", expected, actual)
	}
}

func Test_Test_toString(t *testing.T) {
	mytest := Test{Suite: "Suite", Name: "MyTest"}
	expected := "Suite.MyTest"
	actual := mytest.String()

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func Test_Test_toString_OnlySuite(t *testing.T) {
	mytest := Test{Suite: "Suite"}
	expected := "Suite"
	actual := mytest.String()

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func Test_Test_processResults_test_error_no_results(t *testing.T) {
	mytest := Test{Suite: "Suite", Name: "MyTest"}
	var results []string
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err == nil {
		t.Errorf("Expected error, got none")
	}

	var expectedStatus Status = ERROR
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}

func Test_Test_processResults_test_fail(t *testing.T) {
	mytest := Test{Suite: "DemoSuite", Name: "[test that foo fails]"}
	var results []string
	results = append(results, "[DemoSuite].[test that foo fails] failed: (Failure) Expected: <1> but was: <0>")
	results = append(results, "|Test Execution Summary|")
	results = append(results, "|No|Test Case Name                   |Dur(ms)|Result |")
	results = append(results, "|1 |[DemoSuite].[test that foo fails]|      0|Failure|")
	results = append(results, "Test Case Summary: 1 test case(s) executed, 0 succeeded, 0 skipped, 1 failed, 0 errored.")
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err.Error())
	}

	var expectedStatus Status = FAIL
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}

func Test_Test_processResults_test_error(t *testing.T) {
	mytest := Test{Suite: "DemoSuite", Name: "[test that bar errors]"}
	var results []string
	results = append(results, "[DemoSuite].[test that bar errors] failed: (Error) Message: Divide by zero error encountered. | Procedure: DemoSuite.test that bar errors (4) | Severity, State: 16, 1 | Number: 8134")
	results = append(results, "|Test Execution Summary|")
	results = append(results, "|No|Test Case Name                    |Dur(ms)|Result|")
	results = append(results, "|1 |[DemoSuite].[test that bar errors]|     16|Error |")
	results = append(results, "Test Case Summary: 1 test case(s) executed, 0 succeeded, 0 skipped, 0 failed, 1 errored.")
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err.Error())
	}

	var expectedStatus Status = ERROR
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}

func Test_Test_processResults_test_pass(t *testing.T) {
	mytest := Test{Suite: "DemoSuite", Name: "[test that foo passes]"}
	var results []string
	results = append(results, "|Test Execution Summary|")
	results = append(results, "|No|Test Case Name                    |Dur(ms)|Result |")
	results = append(results, "|1 |[DemoSuite].[test that foo passes]|     16|Success|")
	results = append(results, "Test Case Summary: 1 test case(s) executed, 1 succeeded, 0 skipped, 0 failed, 0 errored.")
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err.Error())
	}

	var expectedStatus Status = PASS
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}

func Test_Test_processResults_test_missing(t *testing.T) {
	mytest := Test{Suite: "DemoSuite", Name: "MyTest"}
	var results []string
	results = append(results, "|Test Execution Summary|")
	results = append(results, "|No|Test Case Name                                                        |Dur(ms)|Result |")
	results = append(results, "Test Case Summary: 0 test case(s) executed, 0 succeeded, 0 skipped, 0 failed, 0 errored.")
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err.Error())
	}

	var expectedStatus Status = MISSING
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}

func Test_Test_processResults_suite_fail(t *testing.T) {
	mytest := Test{Suite: "DemoSuite"}
	var results []string
	results = append(results, "[DemoSuite].[test table_assert] failed: (Failure) Unexpected/missing resultset rows!")
	results = append(results, "|_m_|value|")
	results = append(results, "|=  |True |")
	results = append(results, "|>  |True |")
	results = append(results, "|>  |True |")
	results = append(results, "|>  |True |")
	results = append(results, "|Test Execution Summary|")
	results = append(results, "|No|Test Case Name                  |Dur(ms)|Result |")
	results = append(results, "|1 |[DemoSuite].[test foo passes]   |     94|Success|")
	results = append(results, "|2 |[DemoSuite].[test table_assert] |    312|Failure|")
	results = append(results, "Test Case Summary: 2 test case(s) executed, 1 succeeded, 0 skipped, 1 failed, 0 errored.")
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err.Error())
	}

	var expectedStatus Status = FAIL
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}

func Test_Test_processResults_suite_pass(t *testing.T) {
	mytest := Test{Suite: "DemoSuite"}
	var results []string
	results = append(results, "|Test Execution Summary|")
	results = append(results, "|No|Test Case Name                  |Dur(ms)|Result |")
	results = append(results, "|1 |[DemoSuite].[test foo passes]   |     79|Success|")
	results = append(results, "|2 |[DemoSuite].[test table_assert] |    281|Success|")
	results = append(results, "Test Case Summary: 2 test case(s) executed, 2 succeeded, 0 skipped, 0 failed, 0 errored.")
	mytest.Results = results

	var err error
	mytest.Status, err = mytest.ProcessResults()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err.Error())
	}

	var expectedStatus Status = PASS
	if actualStatus := mytest.Status; actualStatus != expectedStatus {
		t.Errorf("Expected Status: <%s>, got <%s>", expectedStatus, actualStatus)
	}
}
