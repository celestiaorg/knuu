package knuu

import (
	"fmt"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
)

// TestRuns is a struct that contains multiple test runs
type TestRuns struct {
	tests []TestRun
}

// AddTest adds a test run to the test runs
func (t *TestRuns) AddTest(test TestRun) {
	t.tests = append(t.tests, test)
}

// prepareTest prepares a test run
func (t *TestRuns) prepareTest(testRun TestRun, testRunName string) error {
	logrus.Infof("Preparing test run for '%s'", testRunName)
	startTimePrep := time.Now()
	err := testRun.Prepare()
	if err != nil {
		return fmt.Errorf("Error preparing test '%s': %w", testRunName, err)
	}
	endTimePrep := time.Now()
	logrus.Infof("Prepared test run for '%s' in %s", testRunName, endTimePrep.Sub(startTimePrep))
	return nil
}

// runTest runs a test run
func (t *TestRuns) runTest(testRun TestRun, testRunName string) error {
	logrus.Infof("Running test '%s'", testRunName)
	startTimeTest := time.Now()
	err := testRun.Test()
	if err != nil {
		return fmt.Errorf("Error running test '%s': %w", testRunName, err)
	}
	endTimeTest := time.Now()
	logrus.Infof("Ran test '%s' in %s", testRunName, endTimeTest.Sub(startTimeTest))
	return nil
}

// cleanupTest cleans up after a test run
func (t *TestRuns) cleanupTest(testRun TestRun, testRunName string) error {
	logrus.Infof("Cleaning up testrun '%s'", testRunName)
	cleanStartTime := time.Now()
	err := testRun.Clean()
	if err != nil {
		return fmt.Errorf("Error cleaning up after test '%s': %w", testRunName, err)
	}
	cleanEndTime := time.Now()
	logrus.Infof("Cleaned up testrun '%s' in %s", testRunName, cleanEndTime.Sub(cleanStartTime))
	return nil
}

// RunAll runs all the test runs
func (t *TestRuns) RunAll() error {
	logrus.Infof("Running %d test runs", len(t.tests))

	for _, testRun := range t.tests {
		testRunName := reflect.TypeOf(testRun).Elem().Name()

		err := t.prepareTest(testRun, testRunName)
		if err != nil {
			return err
		}

		err = t.runTest(testRun, testRunName)
		if err != nil {
			return err
		}

		err = t.cleanupTest(testRun, testRunName)
		if err != nil {
			return err
		}
	}

	return nil
}
