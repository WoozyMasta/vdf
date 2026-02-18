package vdf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// readFixtureBytes loads one fixture from testdata for tests and benchmarks.
func readFixtureBytes(tb testing.TB, name string) []byte {
	tb.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		tb.Fatalf("failed to read fixture %q: %v", name, err)
	}

	return data
}

// readFixtureString loads one text fixture from testdata.
func readFixtureString(tb testing.TB, name string) string {
	tb.Helper()

	return string(readFixtureBytes(tb, name))
}

// mustReadFixtureBytes loads one fixture for package-level benchmark setup.
func mustReadFixtureBytes(name string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		panic(fmt.Errorf("failed to read fixture %q: %w", name, err))
	}

	return data
}
