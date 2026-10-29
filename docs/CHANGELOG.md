# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.20.0] - 2026-10-29

### Added

- `repo edit [-o OWNER] [flags] <REPO>` (aliases `modify`, `update`): edits a repository; only the flags passed are sent: `--name` (renames it), `--description`, `--website`, `--default-branch`, `--public`, `--archived`, `--template`. Booleans are revoked with `=false`, and an empty string clears a text field. Bitbucket refuses `--archived` and `--template`
- Connections have an optional `owner`: the default owner (user, organization or Bitbucket workspace) of the repositories the `repo` and `branch` commands work on. `conn add` asks for it, `conn edit` changes it (`-` clears it), and `conn ls` / `conn info` show it

### Changed

- `-o`, `--owner` is no longer required by any command. Without it, the owner is the connection's `owner`, else its `username` (unless it is an e-mail), else the account the credentials belong to, as reported by the server. On Bitbucket, where the username is an e-mail and the account name is not necessarily a workspace, the connection needs an `owner` or the command needs `-o`
- `repo list` is unchanged: without `-o`, it still lists all the repositories the user can see
- `branch rm` and `branch mv` on the server now connect before checking the local working copy, which must then be a clone of the owner's repository
- The sample connections file moved to `docs/samples/connections.json`

### Fixed

- Two tests (`TestBranchCommands`, `TestListBranches_Errors`) used the real connections file, and could reach the real server; they now use a throw-away registry

<br><br>
## [0.11.0] - 2026-10-28

### Added

- `branch get -o OWNER [--json] <REPO> <BRANCH>`: shows one branch, from both sides when it exists on both: local commit (and whether it is checked out), server commit, last commit (date, author and title), default and protected flags, and the same status as `branch ls`. A branch on one side only is still shown; outside of a working copy of the repository, only the server's side is shown, with the reason
- `branch rename [-o OWNER] [-l | -r | -a] <REPO> <BRANCH> <NEW NAME>` (alias `mv`): renames a branch locally (the default), on the server, or both (local first). On the server, open pull requests and protection rules follow the branch; with both sides, the local branch then tracks the server's new name. The local rename is done by the git client
- `branch compare -o OWNER [--json] <REPO> <BASE> <HEAD>` (alias `cmp`): counts, on the server, how far ahead of and behind the base the head is (branches, tags or commits), and tells if the head can be merged as a fast-forward
- Tests for `branch rm`, `get`, `rename` and `compare`; those writing to a working copy use a throw-away git repository, and are skipped when the git client is not installed

<br><br>
## [0.10.0] - 2026-10-27

### Added

- `branch ls -o OWNER [-l | -r | -a] <REPO>`: lists the local branches (the working copy in the current directory), the server's, or both (the default), merged by name with a status seen from the local branch: `in sync`, `ahead N` / `behind N` (counted by the server), `unpushed commits`, `local only`, `server only`. The server's side shows each branch's last commit (date and author). The local side is read from the `.git` directory, without the git client
- `branch delete [-o OWNER] [-l | -r | -a] [-f] [-y] <REPO> <BRANCH>...` (aliases `rm`, `del`): deletes branches locally (the default), on the server, or both (local first). The local deletion is done by the git client, which refuses the checked out branch and, unless `-f`, `--force` is passed, an unmerged one. Everything is checked before the confirmation; the run stops at the first failure

### Changed

- Requires gitoperationslib 0.4.0 (`Branch.LastCommit`, `Branches.Compare`)

<br><br>
## [0.9.0] - 2026-10-26

### Added

- `user wipe`: removes a user from all of their organizations (and thus teams), then deletes the user. Everything is checked first, and nothing changes if:
  - the connection's user is not a site administrator, or is the user to wipe
  - the user is the last owner of an organization that still has other members or repositories
  - the user owns repositories, and `--purge` was not passed

  An organization whose only member is the user, and which owns no repositories, is deleted as well. The plan is shown and confirmed once; `-n`, `--dry-run` stops after the plan, `-y`, `--yes` skips the confirmation

<br><br>
## [0.8.2] - 2026-10-26

### Added

- Completed: the `team` subcommand (`list`, `get`, `create`, `edit`, `delete`, `members`, `ismember`, `addmember`, `removemember`, `repos`, `addrepo`, `removerepo`)

### Changed

- Source layout: the feature packages are now grouped by area, under `src/access_mgmt/` (`users`, `organizations`, `teams`) and `src/repo_mgmt/` (`repositories`); `src/branch_mgmt/` and `src/release_mgmt/` are scaffolded for the upcoming branch, tag and release commands

<br><br>
## [0.8.0] - 2026-10-25

### Added

- Completed: the `org` subcommand

### Changed

- Refactored `conn_cfg`, now called `connection`
- Refactored `user`, now called `users`
- `repo create --defaultbranch` renamed to `--default-branch`, like every other multi-word flag

<br><br>
## [0.6.0] - 2026-10-24

### Added

- Completed `user whoami`, `user get` and `user ls`

<br><br>
## [0.5.0] - 2026-10-21

### Added

- Completed `repo create`, finishing all `repo` implementations

<br><br>
## [0.4.0] - 2026-10-21

### Added

- Completed `repo rm`

<br><br>
## [0.3.0] - 2026-10-20

### Added

- Completed `repo info`

<br><br>
## [0.2.0] - 2026-10-20

### Added

- Completed `repo ls`
- Bitbucket Cloud support

### Changed

- Go version bumped to 1.27.3

<br><br>
## [0.1.0] - 2026-10-19

### Added

- Initial packaged version
