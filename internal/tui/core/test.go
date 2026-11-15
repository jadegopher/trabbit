package core

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var (
	update   = flag.Bool("update", false, "update golden files")
	ansiLine = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

func stripANSI(s string) string {
	return ansiLine.ReplaceAllString(s, "")
}

func AssertGolden(t *testing.T, name string, actual string) {
	t.Helper()
	actual = stripANSI(actual)
	goldenPath := filepath.Join("testdata", name+".golden")

	if *update {
		err := os.MkdirAll("testdata", 0755)
		if err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}
		err = os.WriteFile(goldenPath, []byte(actual), 0644)
		if err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v. Run with -update to create it.", err)
	}

	if string(expected) != actual {
		t.Errorf(
			"output does not match golden file %s\n\nEXPECTED:\n%s\n\nACTUAL:\n%s",
			goldenPath,
			string(expected),
			actual,
		)
	}
}
