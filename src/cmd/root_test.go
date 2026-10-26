// gitclt
// Tests the command tree as a whole; also holds execute(), which the other cmd tests use to run a command line

package cmd

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gitclt/organizations"
	"gitclt/testutil"
	"gitclt/users"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// execute runs "gitclt args...", answering prompts with answers; it returns stdout and cobra's own error output
// cobra keeps flag values (and their Changed state) from one Execute() to the next, unlike a real gitclt process
// which runs a single command line: every flag is reset after each run
func execute(t *testing.T, args []string, answers ...string) (string, string) {
	t.Helper()
	defer func() {
		resetFlags(rootCmd)
		// VisibilityFlag refuses its own "" default
		users.CreateUserVisibility, users.EditUserVisibility = "", ""
		organizations.CreateOrgVisibility, organizations.EditOrgVisibility = "", ""
	}()
	var stderr bytes.Buffer
	rootCmd.SetArgs(args)
	rootCmd.SetErr(&stderr)
	out := testutil.Run(t, func() { _ = rootCmd.Execute() }, answers...)
	return out, stderr.String()
}

func resetFlags(c *cobra.Command) {
	for _, fs := range []*pflag.FlagSet{c.Flags(), c.PersistentFlags()} {
		fs.VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// Every command line in the documentation must reach a runnable command, under all its aliases
func TestCommandTree(t *testing.T) {
	for _, line := range []string{
		"version", "whoami", "completion bash", "completion zsh",
		"repo ls", "repository list", "repo info", "repo get", "repo rm", "repo delete", "repo remove",
		"repo create", "repo add",
		"conn ls", "connection list", "conn info", "conn explain", "conn add", "conn create",
		"conn rm", "conn delete", "conn remove", "conn edit", "conn modify", "conn update", "conn default",
		"user whoami", "user who", "user get", "user info", "user ls", "user rm", "user delete", "user remove",
		"user create", "user add", "user edit", "user modify", "user update",
		"org ls", "organization list", "org listall", "org get", "org info", "org rm", "org delete", "org remove",
		"org rename", "org mv", "org create", "org add", "org edit", "org modify", "org update",
		"org members", "org ismember", "org removemember", "org rmmember",
	} {
		c, rest, err := rootCmd.Find(strings.Fields(line))
		if err != nil || len(rest) != 0 || !c.Runnable() {
			t.Errorf("%q: got %q (rest %v), %v", line, c.Name(), rest, err)
		}
	}
}

func TestCommandsAreDocumented(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Short == "" {
			t.Errorf("%s has no Short description", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}

// Every --flag the help text or the examples mention must exist on that command
func TestDocumentedFlagsExist(t *testing.T) {
	flagRef := regexp.MustCompile(`--([a-z][a-z-]*)`)
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, m := range flagRef.FindAllStringSubmatch(c.Long+"\n"+c.Example, -1) {
			if c.Flags().Lookup(m[1]) == nil && c.InheritedFlags().Lookup(m[1]) == nil {
				t.Errorf("%s mentions --%s, which it does not have", c.CommandPath(), m[1])
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}

func TestVersion(t *testing.T) {
	out, _ := execute(t, []string{"version"})
	if !strings.HasPrefix(out, "gitclt v"+buildVersion+" ("+buildDate+"), Go version = v") {
		t.Fatalf("output = %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	if _, stderr := execute(t, []string{"nosuch"}); !strings.Contains(stderr, `unknown command "nosuch"`) {
		t.Fatalf("stderr = %q", stderr)
	}
}

// The persistent flags reach the shared package, from any subcommand
func TestPersistentFlags(t *testing.T) {
	testutil.WriteRegistry(t, "")
	out, _ := execute(t, []string{"conn", "info", "-c", "elsewhere", "-q", "-D", "nosuch"})
	if !strings.Contains(out, "Connection nosuch does not exist") {
		t.Fatalf("output = %q", out)
	}
}

func TestCompleteVisibility(t *testing.T) {
	for _, c := range []*cobra.Command{userCreateCmd, userEditCmd, orgCreateCmd, orgEditCmd} {
		f, ok := c.GetFlagCompletionFunc("visibility")
		if !ok {
			t.Errorf("%s: no completion for --visibility", c.CommandPath())
			continue
		}
		if got, _ := f(c, nil, ""); !slices.Equal(got, []string{"public", "limited", "private"}) {
			t.Errorf("%s: got %v", c.CommandPath(), got)
		}
	}
}
