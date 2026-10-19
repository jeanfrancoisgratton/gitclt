
![gitclt — Manage your Git server, from the CLI](images/gitclt_banner.png)

# gitclt
___
`gitclt` is a CLI client for Forgejo and GitHub (github.com, as well as GitHub Enterprise Server).<br>
It is meant to handle the usual git server tasks from the command line: repositories, branches, tags, releases, users, orgs, etc.<br>
It is a front-end to [gitoperationslib](https://github.com/jeanfrancoisgratton/gitoperationslib), which does the actual talking to the forges. Forgejo servers must run version 11.0.10 or later.

What is currently implemented:
- a connections registry (multiple servers, multiple accounts, one default connection)
- repository listing

**TABLE OF CONTENTS**<br>

[Build / Install requirements](#build-install-requirements)

[Using the tool](#using-the-tool)
- [The connections registry](#connections)
- [Command summary](#command-summary)

[Developer notes: writing a new command](#developer-notes)
- [Package layout](#package-layout)
- [Setting up the client](#http-client)
- [Wiring the command](#wiring-the-command)
- [Testing](#testing)

<a id="build-install-requirements"></a>
# Build/install requirements

## Install from source
1. Clone/fork the repo: either `git clone https://github.com/jeanfrancoisgratton/gitclt` or `git clone https://git.famillegratton.net:3000/devops/gitclt`
2. Ensure that you have the proper GO version, as stated in `src/go.mod`. Run `go version` to check
3. cd to `src`, and then run `./updateBuildDeps.sh`, to ensure that all build dependencies are up to date
4. Run `./build.sh`. By default, the binary will be created in `/opt/bin` (check the directory's permissions ahead of running it). `./build.sh --help` shows the options (binary name, output directory, dry-run)

## Build your own package
All format/distro-specific packaging scripts live under their own directories: `__alpine` (APK), `__debian` (DEB), `__redhat` (RPM), `__archlinux` (PKGBUILD), `__windows`.

<a id="using-the-tool"></a>
# Using the tool

<a id="connections"></a>
## The connections registry
Before doing anything against a server, you need at least one connection. Connections live in `$HOME/.config/JFG/gitclt/connections.json` (see [docs/connections.json](docs/connections.json) for a sample).<br>
The file looks a lot like `$HOME/.docker/config.json`:

```json
{
	"auths": {
		"CONNECTION1": {
			"url": "https://SERVER1:PORT1",
			"token": "...",
			"username": "myuser",
			"password": "...",
			"servertype": "forgejo"
		}
	},
	"default_connection": "CONNECTION1"
}
```

- `servertype` is either `forgejo` or `github`
- `url` is the server URL, as you'd type it in a browser. The API URL is derived from it:
  - Forgejo: `URL/api/v1` (`url` may also be a sub-path install, ex: `https://example.com/forgejo`)
  - github.com: `https://api.github.com`
  - any other host with `servertype: github` is treated as GitHub Enterprise Server: `URL/api/v3`
- You need either a token, or a username + password pair. When both are present, **the token always wins**
- GitHub no longer accepts passwords on its REST API: a GitHub connection needs a token
- Tokens and passwords are encrypted when the tool writes the file. You can still hand-edit the file with cleartext secrets; they will get encrypted the next time the tool saves the registry
- A hand-edited, invalid connection is not rejected on load: `gitclt conn list` flags it, and you can fix it with `gitclt conn edit`, or remove it with `gitclt conn rm`
- The first connection you add becomes the default one

<a id="command-summary"></a>
## Command summary

### Global flags
| Flag            | Meaning                         |
|-----------------|---------------------------------|
| `-q`, `--quiet` | Output will be as quiet as possible |

### Connections: `gitclt conn` (alias: `connection`)
| Command                                   | Description                                                                 |
|-------------------------------------------|-----------------------------------------------------------------------------|
| `gitclt conn list` (`ls`)                 | Lists all connections in the registry                                       |
| `gitclt conn info [CONNECTION...]`        | Shows the details of the given connections; no argument means the default one |
| `gitclt conn add CONNECTION`              | Adds a connection (interactive)                                             |
| `gitclt conn edit CONNECTION`             | Edits a connection in-place; ENTER keeps the current value, `-` clears a secret |
| `gitclt conn rm CONNECTION`               | Removes a connection                                                        |
| `gitclt conn default [CONNECTION]`        | Shows the default connection, or sets it                                    |

### Repositories: `gitclt repo` (alias: `repository`)
All `repo` subcommands accept `-c`, `--connection CONNECTION` to use a given connection instead of the default one.

| Command                                        | Description                                                         |
|------------------------------------------------|---------------------------------------------------------------------|
| `gitclt repo list [-c CONNECTION] [-j]` (`ls`) | Lists all repositories visible to the connection's user. `-j`, `--json` outputs JSON; `-q` outputs only the `owner/name` of each repo |

<a id="developer-notes"></a>
# Developer notes: writing a new command

<a id="package-layout"></a>
## Package layout
`gitclt` is a CLI front-end to [gitoperationslib](https://github.com/jeanfrancoisgratton/gitoperationslib): all forge API calls go through that library.
The code lives under `src/`, and dependencies only flow downward:

```
cmd/            cobra commands; parses flags, calls the feature package, prints the error if any
  │
repositories/   feature packages (one per resource: repositories/, later branches/, releases/, ...)
  │             calls the library, formats the output (table / JSON / quiet)
forge/          opens a gitoperationslib client on a connection; converts library errors
  │
conn_cfg/       the connections registry; resolves a connection name into an ActiveConnection
shared/         global flag values (QuietOutput, ConnectionName)
```

A feature package never reads `connections.json`, never decides how to authenticate, and never sends HTTP requests by itself.
If the library lacks something a command needs, the fix belongs in gitoperationslib, not in `gitclt`.

<a id="http-client"></a>
## Setting up the client

### 1. Connect
```go
ctx := context.Background()
c, err := forge.Connect(ctx, shared.ConnectionName)
if err != nil {
	return err
}
```
`forge.Connect()` takes a connection name; an empty name means the default connection. Always pass
`shared.ConnectionName`, which holds the `-c`/`--connection` flag value. It returns a `*cerr.CustomError`, so you can return it as-is.

Behind the scenes:
- `conn_cfg.ResolveConnection()` loads the registry, picks the connection (and errors out if it's unknown, invalid, or if there's no default connection), and decides the authentication method (token over username/password)
- `forge.Connect()` then opens the matching library client:
  - Forgejo: `forgejo.New()` with `forgejo.Token()` or `forgejo.BasicAuth()`. **This already contacts the server**: it reads its version, and refuses anything older than the library's minimum (11.0.10)
  - GitHub: `github.New()` with the token; any host other than `github.com` is treated as GitHub Enterprise Server (`github.WithEnterpriseURL()`, the library appends `/api/v3/`). This makes no request
- both get an `http.Client` with connect / TLS handshake / response headers timeouts (`forge.FastFailTimeout`, 30s by default), and `gitclt` as User-Agent. There is no global request timeout, so long transfers (ex: release assets) are not cut; use a context if you need a deadline

You get back a `*forge.Client`:
- it embeds `gol.Client` (`gol` being the usual alias for `github.com/jeanfrancoisgratton/gitoperationslib`), so the library's areas are called directly on it: `c.Repos()`, `c.Branches()`, `c.Tags()`, `c.Releases()`, `c.PullRequests()`, `c.Issues()`, `c.Labels()`, `c.Milestones()`, `c.Webhooks()`, ...
- `c.Connection` is the resolved `conn_cfg.ActiveConnection`: its `Name` is handy in output messages, its `Username` in user-scoped calls

See the [gitoperationslib README](https://github.com/jeanfrancoisgratton/gitoperationslib#readme) for the full API of each area.

### 2. Errors
The library returns plain `error`s; commands return `*cerr.CustomError`. Convert with `forge.Error(title, err)`, which also copies the HTTP status (if any) into `Code`:

```go
repo, gerr := c.Repos().Get(ctx, owner, name)
if gerr != nil {
	return forge.Error("Unable to fetch the repository", gerr)
}
```

Test the library error **before** converting it, when one case is normal for your command:
```go
repo, gerr := c.Repos().Get(ctx, owner, name)
switch {
case gol.IsNotFound(gerr):
	// no such repository
case errors.Is(gerr, gol.ErrNotSupported):
	// this backend cannot do this
case gerr != nil:
	return forge.Error("Unable to fetch the repository", gerr)
}
```
Also available: `gol.IsUnauthorized()`, `gol.IsForbidden()`, `gol.StatusCode()`, and `*gol.APIError` (via `errors.As`) for the forge's message and details.

### 3. Lists: always iterate with `gol.All()`
List methods return **one page** (`gol.Page[T]`). Calling them once only gives you the first page, without any warning. Use `gol.All()`, which follows the pages for you:

```go
var repos []gol.Repo
fetch := func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
	return c.Repos().List(ctx, "", o)
}
for r, lerr := range gol.All(ctx, gol.ListOptions{PerPage: listPageSize}, fetch) {
	if lerr != nil {
		return forge.Error("Unable to list repositories", lerr)
	}
	repos = append(repos, r)
}
```
- the iteration stops at the first error, which is yielded with a zero item: check `lerr` before using the item
- `PerPage`: 50 is safe on both forges (Forgejo's maximum is a server setting, 50 by default; GitHub's is 100)

### 4. Creating, editing, deleting
The library's option structs use the library's own types; nil pointer fields in `Edit...Options` mean "leave unchanged" (`gol.Ptr()` builds a pointer from a value):

```go
repo, gerr := c.Repos().Create(ctx, gol.CreateRepoOptions{Name: name, Visibility: gol.VisibilityPrivate})
if gerr != nil {
	return forge.Error("Unable to create the repository", gerr)
}

_, gerr = c.Repos().Edit(ctx, owner, name, gol.EditRepoOptions{Archived: gol.Ptr(true)})

gerr = c.Repos().Delete(ctx, owner, name)
```

A deadline on a single operation:
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
```

### 5. Forge differences
The library hides most differences behind its interfaces. When a feature only exists on one backend:
- `c.Capabilities().Has(gol.CapAdmin)` (or a type assertion such as `c.Client.(gol.Admin)`) tells you whether it's available
- otherwise, the call returns `gol.ErrNotSupported`: report it, don't work around it
- `c.Backend()` returns `gol.BackendForgejo` or `gol.BackendGitHub`

Some fields are not reported by every backend (ex: `gol.Repo.Empty` is always false on GitHub); check the library's doc comments before relying on one.

<a id="wiring-the-command"></a>
## Wiring the command
- The feature function returns a `*cerr.CustomError`; the cobra command prints it (see `cmd/repo_commands.go`)
- Honour the output flags: `shared.QuietOutput` (bare output, one item per line), and a `--json` flag when it makes sense. The JSON output is the library's types as-is (Go field names, ex: `FullName`)
- `-c`/`--connection` is a persistent flag on the `repo` command group. A new command group (ex: `branch`) must declare it too:
  ```go
  branchCmd.PersistentFlags().StringVarP(&shared.ConnectionName, "connection", "c", "", "Connection to use (default: the default connection)")
  ```
  It is deliberately not on the root command: the `conn` commands take connection names as arguments

<a id="testing"></a>
## Testing
`forge/client_test.go` shows the pattern: an in-process `httptest` server stands in for the forge, and the registry is a temporary one.
- point `HOME` to a temporary directory (`t.Setenv("HOME", t.TempDir())`), and write a `.config/JFG/gitclt/connections.json` there, with the `httptest` server URL as `url` and a cleartext token (see `writeRegistry()`)
- then call `forge.Connect()` as usual
- a `forgejo` test server must answer `GET /api/v1/version` (ex: `{"version":"15.0.2"}`), since `forgejo.New()` checks it; the other calls are under `/api/v1/`
- a `github` test server is treated as GitHub Enterprise Server: the calls are under `/api/v3/`

`build.sh` runs `go vet ./...` and `go test ./...` before building.
