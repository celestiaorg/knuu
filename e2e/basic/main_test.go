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

func (s *TestSuite) TestMain() {
	s.T().Parallel()
	// Setup

	// Test Logic
	s.T().Log("Running test case: TestMain")

	// Perform the test
	exitVal := s.Run("TestMain", func() {
		s.T().Logf("Scope: %s", s.Knuu.Scope())
	})

	exitValue := convertViaMap(exitVal)

	os.Exit(exitValue)
}
