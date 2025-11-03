package minikanren

import "os"

// shouldRunHeavy returns true when heavy/long-running tests should run even
// if the Go test suite is invoked in short mode. Set GOKANDO_FORCE_HEAVY=1
// (or "true") to override short-mode skips.
func shouldRunHeavy() bool {
	v := os.Getenv("GOKANDO_FORCE_HEAVY")
	return v == "1" || v == "true" || v == "TRUE" || v == "True"
}
