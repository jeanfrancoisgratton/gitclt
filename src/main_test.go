// gitclt
// Tests main(): the config directory is created, the command runs, and the working directory is restored

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestMain_Version(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	testutil.Set(t, &os.Args, []string{"gitclt", "version"})
	wd, _ := os.Getwd()

	out := testutil.Run(t, main)
	if !strings.HasPrefix(out, "gitclt v") {
		t.Fatalf("output = %q", out)
	}
	if fi, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".config", "JFG", "gitclt")); err != nil || !fi.IsDir() {
		t.Fatalf("config directory: %v", err)
	}
	if now, _ := os.Getwd(); now != wd {
		t.Fatalf("working directory changed to %s", now)
	}
}
