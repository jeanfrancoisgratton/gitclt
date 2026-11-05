// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/forge/otp.go
// Original timestamp : 2026.11.04 23:30:00

package forge

import (
	"context"
	"errors"
	"fmt"
	"gitclt/connections"
	"gitclt/shared"
	"os"
	"strings"
	"sync"
	"time"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/forgejo"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"golang.org/x/term"
)

// Forgejo accepts a code until the end of the 30-second period after the one it was generated in
const otpPeriod = 30 * time.Second

// A code received this early in a period may have been read just before the period started: it then lasts one period less
const otpGrace = 10 * time.Second

const otpAttempts = 3

// Replaced by the tests
var otpClock = time.Now

// A code stays good for at least this long after being received
func otpExpiry(received time.Time) time.Time {
	start := received.Truncate(otpPeriod)
	if received.Sub(start) < otpGrace {
		return start.Add(otpPeriod)
	}
	return start.Add(2 * otpPeriod)
}

// GetStringValFromPrompt would otherwise block on, or read from, whatever stdin is; like the password prompts,
// debug mode lets it read from a pipe
func canPromptOTP() bool {
	return shared.DebugMode || term.IsTerminal(int(os.Stdin.Fd()))
}

func promptOTP(username string) string {
	return strings.TrimSpace(hf.GetStringValFromPrompt("Enter the two-factor authentication code of " + username + ": "))
}

// otpSource gives the library the current code before each request: a long run outlasts a code, so a new one is asked for
// when it expires; without a terminal, the old code is kept, and the server's refusal ends the run
type otpSource struct {
	mu       sync.Mutex
	username string
	code     string
	expires  time.Time
}

func newOTPSource(username, code string) *otpSource {
	return &otpSource{username: username, code: code, expires: otpExpiry(otpClock())}
}

// The library may call this from several goroutines: they all wait for the same answer
func (o *otpSource) get() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if otpClock().Before(o.expires) || !canPromptOTP() {
		return o.code
	}
	if code := promptOTP(o.username); code != "" {
		o.code = code
		o.expires = otpExpiry(otpClock())
	}
	return o.code
}

// Opens a Forgejo client with a username and password, for an account that may have two-factor authentication:
// --otp when passed, else the code is only asked for once the server says it needs one (New's version check is enough
// to tell), so that other accounts never see the question
func connectForgejoPassword(ctx context.Context, ac connections.ActiveConnection, username, password string) (gol.Client, *cerr.CustomError) {
	open := func(creds forgejo.Credentials) (gol.Client, error) {
		return forgejo.New(ctx, ac.URL, creds, forgejo.WithHTTPClient(httpClient()), forgejo.WithUserAgent(userAgent))
	}
	failed := func(e error, hint string) *cerr.CustomError {
		ce := Error(fmt.Sprintf("Unable to connect to %s", ac.Name), e)
		ce.Message += hint
		return ce
	}
	withCode := func(code string) (gol.Client, error) {
		return open(forgejo.BasicAuthOTPFunc(username, password, newOTPSource(username, code).get))
	}

	if shared.OTP != "" {
		gc, e := withCode(shared.OTP)
		if errors.Is(e, gol.ErrInvalidOTP) {
			return nil, failed(e, "; the code passed with --otp is wrong, or has expired")
		}
		if e != nil {
			return nil, failed(e, "")
		}
		return gc, nil
	}

	gc, e := open(forgejo.BasicAuth(username, password))
	if e != nil && !errors.Is(e, gol.ErrInvalidOTP) {
		return nil, failed(e, "")
	}
	if e == nil {
		return gc, nil
	}
	if !canPromptOTP() {
		return nil, failed(e, "; pass the current code with --otp")
	}
	for range otpAttempts {
		code := promptOTP(username)
		if code == "" {
			return nil, failed(e, "; no code was entered")
		}
		if gc, e = withCode(code); e == nil {
			return gc, nil
		}
		if !errors.Is(e, gol.ErrInvalidOTP) {
			return nil, failed(e, "")
		}
		fmt.Println(hftx.WarningSign("This code is wrong, or has expired"))
	}
	return nil, failed(e, "")
}
