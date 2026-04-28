package validate

import (
	"errors"
	"os"
)

// ErrFixtureNotFound is returned by readFixture when none of the candidate
// paths resolve. Test-only helper; lives in the package so tests can import
// it without an internal/ split.
var ErrFixtureNotFound = errors.New("fixture not found in any expected location")

func readFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
