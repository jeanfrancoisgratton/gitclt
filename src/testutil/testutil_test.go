// gitclt
// Tests the test helpers themselves: if Run fed the prompts wrongly, every interactive test would be lying

package testutil

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
)

func TestRun_AnswersEachPromptInTurn(t *testing.T) {
	var a, b, c string
	out := Run(t, func() {
		fmt.Println("some output first")
		a = hf.GetStringValFromPrompt("first: ")
		b = hf.GetStringValFromPrompt("second: ")
		c = hf.GetPassword("third: ", true)
	}, "one", "", "three")

	if a != "one" || b != "" || c != "three" {
		t.Fatalf("got %q, %q, %q", a, b, c)
	}
	if !strings.Contains(out, "some output first") || !strings.Contains(out, "third: ") {
		t.Fatalf("output = %q", out)
	}
}

// A line written in pieces ("key": then value) must not be mistaken for a prompt
func TestRun_PiecemealOutputIsNotAPrompt(t *testing.T) {
	out := Run(t, func() {
		fmt.Print(`"id": `)
		fmt.Print("42\n")
	})
	if out != "\"id\": 42\n" {
		t.Fatalf("got %q", out)
	}
}

func TestRun_StripsColours(t *testing.T) {
	if out := Run(t, func() { fmt.Println("\x1b[32mgreen\x1b[0m") }); out != "green\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSet_RestoresTheValue(t *testing.T) {
	v := "before"
	t.Run("inner", func(t *testing.T) {
		Set(t, &v, "during")
		if v != "during" {
			t.Fatalf("got %q", v)
		}
	})
	if v != "before" {
		t.Fatalf("got %q", v)
	}
}

func TestForgejo_RoutesAndRegistry(t *testing.T) {
	srv := Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/ping": Reply(`{"ok":true}`)})

	for path, want := range map[string]int{"/api/v1/version": 200, "/api/v1/ping": 200, "/api/v1/nope": 404} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("%s: status %d, want %d", path, resp.StatusCode, want)
		}
	}

	b, err := os.ReadFile(os.Getenv("HOME") + "/.config/JFG/gitclt/connections.json")
	if err != nil || !strings.Contains(string(b), srv.URL) {
		t.Fatalf("registry = %s, %v", b, err)
	}
}

func TestRecord(t *testing.T) {
	var bodies []map[string]any
	srv := Forgejo(t, map[string]http.HandlerFunc{
		"POST /api/v1/ok":  Record(t, &bodies, http.StatusCreated, `{"id":1}`),
		"POST /api/v1/bad": Record(t, &bodies, http.StatusConflict, ""),
	})

	for path, want := range map[string]struct {
		status int
		body   string
	}{"/api/v1/ok": {201, `{"id":1}`}, "/api/v1/bad": {409, `{"message":"Conflict"}`}} {
		resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader(`{"name":"acme"}`))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != want.status || string(b) != want.body {
			t.Errorf("%s: %d %s", path, resp.StatusCode, b)
		}
	}
	if len(bodies) != 2 || bodies[0]["name"] != "acme" || bodies[1]["name"] != "acme" {
		t.Fatalf("recorded %v", bodies)
	}
}
