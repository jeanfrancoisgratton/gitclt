// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/testutil/testutil.go
// Original timestamp : 2026.10.24 21:00:00

// Package testutil holds what the _test.go files of the other packages share: a throw-away connections registry,
// a fake Forgejo server, and a way to answer the interactive prompts while capturing stdout
// It is only ever imported from _test.go files, so it never ends up in the gitclt binary
package testutil

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Set changes *p to v for the duration of the test; the previous value is restored afterwards
// Use it on the package-level flag variables (JSONoutput, AssumeYes...), so that one test cannot leak into the next
func Set[T any](t *testing.T, p *T, v T) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

// WriteRegistry points HOME to a temporary directory, and writes registry as its connections.json
// It returns the path of the file
func WriteRegistry(t *testing.T, registry string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dir := filepath.Join(os.Getenv("HOME"), ".config", "JFG", "gitclt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "connections.json")
	if registry != "" {
		if err := os.WriteFile(path, []byte(registry), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// Forgejo starts a fake Forgejo server, and makes it the default connection (named "test", token "abc")
// routes are net/http patterns, ex: "GET /api/v1/users/{name}"; the version check is already answered
// Anything else gets a 404, which is also how "no such user / org / repo" is simulated
func Forgejo(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", Reply(`{"version":"15.0.2"}`))
	for pattern, h := range routes {
		mux.HandleFunc(pattern, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	WriteRegistry(t, `{"auths":{"test":{"url":"`+srv.URL+`","token":"abc","servertype":"forgejo"}},"default_connection":"test"}`)
	return srv
}

// Reply answers 200 with a JSON body
func Reply(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
}

// Status answers an empty response with the given HTTP status; 204 is what Forgejo sends on a successful DELETE
func Status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if code >= 400 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			fmt.Fprintf(w, `{"message":"%s"}`, http.StatusText(code))
			return
		}
		w.WriteHeader(code)
	}
}

var ansi = regexp.MustCompile("\x1b\\[[0-9;]*m")

// StripANSI removes the colour codes terminalfx adds, so tests can look for plain text
func StripANSI(s string) string {
	return ansi.ReplaceAllString(s, "")
}

// How long the output must stay still, after a line ending with ": ", for it to be taken as a prompt
const promptQuiet = 25 * time.Millisecond

// Run calls f with stdout captured, and returns what it printed (without the colour codes)
// Every prompt f shows is answered with the next of answers. A prompt is recognised as output ending with ": "
// that is not followed by anything else for a short while: helperFunctions prints it, then blocks on stdin
// (the "nothing follows" part matters: some printers, like prettyjson, write a line in several pieces)
// Each answer is only written once its prompt is up, as helperFunctions creates a new bufio.Scanner for each prompt:
// if all answers were written upfront, the first scanner would swallow them all
// A prompt with no answer left gets EOF (an empty answer), and fails the test, as does an unused answer
func Run(t *testing.T, f func(), answers ...string) string {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = inR, outW
	defer func() {
		os.Stdin, os.Stdout = oldIn, oldOut
		_ = inR.Close()
		_ = outR.Close()
	}()

	chunks := make(chan []byte)
	go func() {
		defer close(chunks)
		buf := make([]byte, 64*1024)
		for {
			n, rerr := outR.Read(buf)
			if n > 0 {
				chunks <- bytes.Clone(buf[:n])
			}
			if rerr != nil {
				return
			}
		}
	}()

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer inW.Close()
		quiet := time.NewTimer(time.Hour)
		defer quiet.Stop()
		for {
			select {
			case c, ok := <-chunks:
				if !ok {
					return
				}
				out.Write(c)
				quiet.Reset(promptQuiet)
			case <-quiet.C:
				line := StripANSI(out.String())
				if !strings.HasSuffix(line, ": ") {
					continue
				}
				if len(answers) == 0 {
					t.Errorf("unexpected prompt, no answer left: %q", line[strings.LastIndex(line, "\n")+1:])
					_ = inW.Close()
					continue
				}
				fmt.Fprintln(inW, answers[0])
				answers = answers[1:]
			}
		}
	}()

	func() {
		defer outW.Close()
		f()
	}()
	<-done
	if len(answers) > 0 {
		t.Errorf("%d answer(s) not used: %q", len(answers), answers)
	}
	return StripANSI(out.String())
}
