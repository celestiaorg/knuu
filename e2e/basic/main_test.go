package basic

import (
	"os"
)

func convertViaMap(b bool) int {
	table := map[bool]int{
		true:  1,
		false: 0,
	}
	return table[b]
}

func (ts *TestSuite) TestMain() {
	ts.T().Parallel()
	// Setup

	// Test Logic
	ts.T().Log("Running test case: TestMain")

	// Perform the test
	exitVal := ts.Run("TestMain", func() {
		s.T().Logf("Scope: %s", ts.Knuu.Scope())
	})

	exitValue := convertViaMap(exitVal)

	os.Exit(exitValue)
}
