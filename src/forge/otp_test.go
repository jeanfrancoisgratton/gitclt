// gitclt
// Tests two-factor authentication on password connections, against a fake Forgejo account that has it enabled

package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"gitclt/shared"
	"gitclt/testutil"
)

// Prompts need a terminal, unless in debug mode: that is how Run's answers get read
func otpDefaults(t *testing.T) {
	t.Helper()
	testutil.Set(t, &shared.OTP, "")
	testutil.Set(t, &shared.DebugMode, false)
	testutil.Set(t, &shared.ConnectionName, "")
}

// Every request needs a valid code unless the account has no 2FA; seen collects the header of each request
type otpServer struct {
	mu   sync.Mutex
	seen []string
	auth []string
	url  string
}

func startOTPServer(t *testing.T, twoFactor bool, valid ...string) *otpServer {
	t.Helper()
	s := &otpServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		code := r.Header.Get("X-FORGEJO-OTP")
		s.seen = append(s.seen, code)
		s.auth = append(s.auth, strings.Fields(r.Header.Get("Authorization"))[0])
		w.Header().Set("Content-Type", "application/json")
		if twoFactor && s.auth[len(s.auth)-1] == "Basic" && !slices.Contains(valid, code) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"message":"invalid provided OTP"}`)
			return
		}
		if r.URL.Path == "/api/v1/version" {
			fmt.Fprint(w, `{"version":"15.0.2"}`)
			return
		}
		fmt.Fprint(w, `{"id":1,"login":"me","name":"web","full_name":"me/web","owner":{"login":"me"}}`)
	}))
	t.Cleanup(srv.Close)
	s.url = srv.URL
	writeRegistry(t, `{"url":"`+srv.URL+`","username":"me","password":"pw","servertype":"forgejo"}`)
	return s
}

func connectAnswering(t *testing.T, answers ...string) (*Client, string, error) {
	t.Helper()
	var c *Client
	var err error
	out := testutil.Run(t, func() {
		cl, ce := Connect(context.Background(), "")
		c = cl
		if ce != nil {
			err = ce
		}
	}, answers...)
	return c, out, err
}

func TestOTP_AccountWithoutIsNeverAsked(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.DebugMode, true)
	s := startOTPServer(t, false)
	_, out, err := connectAnswering(t)
	if err != nil || out != "" || s.seen[0] != "" {
		t.Fatalf("%v, output = %q, seen = %q", err, out, s.seen)
	}
}

func TestOTP_Flag(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.OTP, "123456")
	s := startOTPServer(t, true, "123456")
	_, out, err := connectAnswering(t)
	if err != nil || out != "" || !slices.Equal(s.seen, []string{"123456"}) {
		t.Fatalf("%v, output = %q, seen = %q", err, out, s.seen)
	}
}

// A wrong --otp is reported; no second chance, as it may be a script
func TestOTP_WrongFlag(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.DebugMode, true)
	testutil.Set(t, &shared.OTP, "000000")
	startOTPServer(t, true, "123456")
	_, _, err := connectAnswering(t)
	if err == nil || !strings.Contains(err.Error(), "wrong, or has expired") {
		t.Fatalf("got %v", err)
	}
}

func TestOTP_NoTerminal(t *testing.T) {
	otpDefaults(t)
	startOTPServer(t, true, "123456")
	_, out, err := connectAnswering(t)
	if err == nil || !strings.Contains(err.Error(), "pass the current code with --otp") || out != "" {
		t.Fatalf("%v, output = %q", err, out)
	}
}

func TestOTP_Prompt(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.DebugMode, true)
	s := startOTPServer(t, true, "123456")
	_, out, err := connectAnswering(t, " 123456 ")
	if err != nil || !strings.Contains(out, "two-factor authentication code of me") {
		t.Fatalf("%v, output = %q", err, out)
	}
	// First without a code, to find out that one is needed
	if !slices.Equal(s.seen, []string{"", "123456"}) {
		t.Fatalf("seen = %q", s.seen)
	}
}

func TestOTP_PromptRetries(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.DebugMode, true)
	startOTPServer(t, true, "123456")
	_, out, err := connectAnswering(t, "000000", "123456")
	if err != nil || !strings.Contains(out, "wrong, or has expired") {
		t.Fatalf("%v, output = %q", err, out)
	}

	_, out, err = connectAnswering(t, "1", "2", "3")
	if err == nil || strings.Count(out, "wrong, or has expired") != 3 {
		t.Fatalf("%v, output = %q", err, out)
	}

	if _, _, err = connectAnswering(t, ""); err == nil || !strings.Contains(err.Error(), "no code was entered") {
		t.Fatalf("got %v", err)
	}
}

// A token needs no code: nothing is asked, and --otp changes nothing
func TestOTP_TokenIsNotAffected(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.DebugMode, true)
	s := startOTPServer(t, true, "123456")
	writeRegistry(t, `{"url":"`+s.url+`","username":"me","password":"pw","token":"abc","servertype":"forgejo"}`)
	for _, flag := range []string{"", "000000"} {
		testutil.Set(t, &shared.OTP, flag)
		_, out, err := connectAnswering(t)
		if err != nil || out != "" {
			t.Fatalf("--otp %q: %v, output = %q", flag, err, out)
		}
	}
	for i, code := range s.seen {
		if code != "" || s.auth[i] == "Basic" {
			t.Fatalf("seen = %q, auth = %q", s.seen, s.auth)
		}
	}
}

func TestOTPExpiry(t *testing.T) {
	base := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ at, want time.Duration }{
		{0, 30 * time.Second},                // may have been read in the previous period
		{9 * time.Second, 30 * time.Second},  // same
		{10 * time.Second, 60 * time.Second}, // read in this period
		{29 * time.Second, 60 * time.Second}, // same
		{30 * time.Second, 60 * time.Second}, // next period, early
		{45 * time.Second, 90 * time.Second}, // next period, late
	} {
		if got := otpExpiry(base.Add(tc.at)); !got.Equal(base.Add(tc.want)) {
			t.Errorf("at %v: got %v", tc.at, got.Sub(base))
		}
	}
}

// A run outlasting its code asks for a new one, once; the code is reused until it expires
func TestOTP_ExpiresDuringARun(t *testing.T) {
	otpDefaults(t)
	testutil.Set(t, &shared.DebugMode, true)
	now := time.Date(2026, 11, 4, 12, 0, 20, 0, time.UTC)
	testutil.Set(t, &otpClock, func() time.Time { return now })
	s := startOTPServer(t, true, "111111", "222222")

	c, _, err := connectAnswering(t, "111111")
	if err != nil {
		t.Fatal(err)
	}
	get := func(answers ...string) string {
		t.Helper()
		out := testutil.Run(t, func() {
			if _, e := c.Repos().Get(context.Background(), "me", "web"); e != nil {
				t.Error(e)
			}
		}, answers...)
		return out
	}

	now = now.Add(20 * time.Second) // 12:00:40, the code is good until 12:01:00
	if out := get(); out != "" {
		t.Errorf("asked too early: %q", out)
	}
	now = now.Add(25 * time.Second) // 12:01:05
	if out := get("222222"); !strings.Contains(out, "two-factor authentication code of me") {
		t.Errorf("not asked: %q", out)
	}
	if out := get(); out != "" {
		t.Errorf("asked twice: %q", out)
	}
	if last := s.seen[len(s.seen)-1]; last != "222222" {
		t.Errorf("seen = %q", s.seen)
	}
}

// Without a terminal, the code is kept, and the server's refusal ends the run
func TestOTP_ExpiresWithoutTerminal(t *testing.T) {
	otpDefaults(t)
	now := time.Date(2026, 11, 4, 12, 0, 20, 0, time.UTC)
	testutil.Set(t, &otpClock, func() time.Time { return now })
	testutil.Set(t, &shared.OTP, "111111")
	s := startOTPServer(t, true, "111111")

	c, _, err := connectAnswering(t)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(5 * time.Minute)
	out := testutil.Run(t, func() {
		if _, e := c.Repos().Get(context.Background(), "me", "web"); e != nil {
			t.Error(e)
		}
	})
	if last := s.seen[len(s.seen)-1]; out != "" || last != "111111" {
		t.Errorf("output = %q, seen = %q", out, s.seen)
	}
}
