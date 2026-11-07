# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - 2026-11-07

### Added

- `run` commands (alias `runs`): CI workflow runs, on Forgejo Actions and GitHub Actions; Bitbucket Pipelines is not supported yet. On Forgejo, some need 16 or 17, and say so on an older server. See "CI workflow runs" in the README:
  - `run list [-w WORKFLOW] [-s STATUS] [-e EVENT] [-b BRANCH] [--sha SHA] [-n COUNT] [-j]` (alias `ls`): lists the runs, newest first; `-q` outputs only their IDs
  - `run get [-j] <RUN-ID>` (aliases `info`, `show`): shows a run, with its jobs
  - `run cancel <RUN-ID>...` (alias `stop`), `run rerun <RUN-ID>`, `run delete [-y] <RUN-ID>...` (aliases `rm`, `del`): cancel skips finished runs, delete refuses one still going
  - `run logs [-O FILE] [--force] <RUN-ID>`: saves the logs of a run's jobs, as a ZIP archive
  - `run job list|get|rerun|log` (alias `jobs`): a run's jobs, with their steps; `log` shows a job's log as text
  - `run artifact list|get|download|delete` (alias `artifacts`): the files runs store; `download` saves one as a ZIP archive
- `issue labels` lists every label an issue can take: the repository's and, on Forgejo, its organization's, with a scope column and a swatch of each color
- `issue label list|get|create|edit|delete --org ORG`: an organization's labels (Forgejo only)
- `issue label get|edit|delete --id ID`: designate a label by the ID the list shows, where Forgejo let several share a name

### Changed

- Built on gitoperationslib 1.6.0, which now handles the issue traps gitclt used to work around itself: pull request numbers, label and milestone names in use, unknown label filters, GitHub's deleted issues and its silent label creation
- `issue milestone edit --due ""` now removes the due date on GitHub; Forgejo's API still cannot, and the command says so
- `issue label edit` and `delete` refuse a name several labels share, and point to `--id`, instead of only refusing

## [1.5.1] - 2026-11-06

### Changed

- foreground + background colours in multiple `ls` commands (_eg: gitclt token ls_)

## [1.5.0] - 2026-11-06

### Added

- `secret` commands (alias `secrets`): CI secrets, for Forgejo Actions, GitHub Actions and Bitbucket Pipelines. They are write-only: they can be replaced or deleted, never read back. See "CI secrets" in the README:
  - `secret list [-j]` (alias `ls`): lists their names; `-q` outputs only those
  - `secret set [-f FILE | --stdin] <NAME>` (aliases `add`, `create`, `update`): creates a secret, or replaces its value without asking. The value is never an argument: it is prompted for without being shown, read from a file only you can access, or from the standard input
  - `secret delete [-y] <NAME>...` (aliases `rm`, `remove`, `del`): deletes secrets, after confirmation
- `variable` commands (aliases `variables`, `var`, `vars`): CI variables, like secrets but readable:
  - `variable list [-j]` (alias `ls`): lists them, with the beginning of their values; `-q` outputs only their names
  - `variable get [-j] <NAME>` (alias `info`): shows a variable, with its whole value; `-q` outputs only the value
  - `variable set <NAME> <VALUE>` (aliases `add`, `create`, `update`): creates a variable, or changes its value; `-f`, `--from-file` or `--stdin` replace `<VALUE>`
  - `variable delete [-y] <NAME>...` (aliases `rm`, `remove`, `del`): deletes variables, after confirmation
- Secrets and variables belong to a repository, or with `--org ORG` to an organization or Bitbucket workspace, with `--user` to your own account (Forgejo only), with `--env ENV` to an environment of the repository (GitHub, Bitbucket)
- `collaborator` commands (aliases `collaborators`, `collab`, `collabs`): the users with their own access to a repository, on Forgejo, GitHub and Bitbucket. See "Collaborators" in the README:
  - `collaborator list [-j]` (alias `ls`): lists them; `-q` outputs only their logins
  - `collaborator get [-j] <USER>` (alias `info`): shows a collaborator's access
  - `collaborator add [-p PERMISSION] <USER>` (aliases `create`, `set`, `update`): gives a user access, or changes that of a collaborator; it tells when GitHub only invited them
  - `collaborator remove [-y] <USER>...` (aliases `rm`, `delete`, `del`): takes access away, after confirmation; removing someone who has none is an error here, not a silent success. On GitHub, it also withdraws a pending invitation
  - `collaborator invitations [-j]` (alias `invites`): lists who has not accepted yet (GitHub only)
- All of these work from inside a clone, without `<REPO>` nor `-o`, like the other repository commands

### Changed

- GitHub's issue lists can take a few seconds to catch up with a change: `issue create` now says so in its help

### Fixed

- `issue create` and `issue edit` no longer create a label on GitHub: GitHub does it without a word for a name it does not know, so a typo left a new label behind. They now refuse, as Forgejo does
- `issue milestone` showed a GitHub due date one day early west of Greenwich, as GitHub only keeps the day
- `issue get`, `issue edit` and `issue close` on a deleted GitHub issue now say that there is no such issue, instead of showing the HTTP 410

## [1.4.0] - 2026-11-06

### Added

- `user email` commands (alias `emails`): the e-mail addresses of the connection's user, on Forgejo and GitHub (Bitbucket can only list them):
  - `user email list [-j]` (alias `ls`): lists the addresses; `-q` outputs only the addresses
  - `user email add [-j] <ADDRESS>...` (alias `create`): adds addresses
  - `user email delete [-y] <ADDRESS>...` (aliases `rm`, `remove`): removes addresses, after confirmation; nothing changes if one is unknown, or is the primary address
- `user sshkey` commands (aliases `sshkeys`, `ssh`): the public SSH keys of the connection's user, on Forgejo, GitHub and Bitbucket:
  - `user sshkey list [-j]` (alias `ls`): lists the keys; `-q` outputs only their titles
  - `user sshkey get [-j] <TITLE>` or `--id ID` (alias `info`): shows a key
  - `user sshkey add [-j] <TITLE> <PUBLIC-KEY-FILE>` (alias `create`): adds a key; anything but one public key, such as a private key, is refused before contacting the server
  - `user sshkey delete [-y] <TITLE>` or `--id ID` (aliases `rm`, `remove`): removes a key, after confirmation; a title shared by several keys is refused, and needs `--id`
- `user avatar` commands: the avatar of the connection's user, on Forgejo (GitHub and Bitbucket have no API for this):
  - `user avatar set <IMAGE-FILE>` (alias `upload`): uploads an image, of at most 10 MiB and 4096x4096
  - `user avatar delete [-y]` (aliases `rm`, `remove`): restores the default avatar, after confirmation
- `org avatar` commands: the avatar of an organization, on Forgejo, for its owners and site administrators; a token needs the `write:organization` scope:
  - `org avatar set <ORG> <IMAGE-FILE>` (alias `upload`): uploads an image, with the same limits as `user avatar set`
  - `org avatar delete [-y] <ORG>` (aliases `rm`, `remove`): restores the default avatar, after confirmation
- `issue` commands (alias `issues`), on Forgejo and GitHub, as Bitbucket Cloud retired its issue tracker. See "Issues" in the README:
  - `issue list [-s open|closed|all] [-l LABEL,...] [-j]` (alias `ls`): lists the issues, without the pull requests; `-q` outputs only their numbers
  - `issue get [--comments] [-j] <NUMBER>` (alias `info`): shows an issue, with its text
  - `issue create -t TITLE [flags]` (aliases `add`, `open`): opens an issue; `-q` outputs only its number
  - `issue edit [flags] <NUMBER>` (aliases `modify`, `update`): only changes what the flags passed say; `--add-label` and `--remove-label` change labels one by one
  - `issue close <NUMBER>...`, `issue reopen <NUMBER>...`: change the state; those already in that state are skipped
  - `issue delete [-y] <NUMBER>...` (aliases `rm`, `del`): deletes issues for good, after confirmation
  - `issue lock [-r REASON] <NUMBER>`, `issue unlock <NUMBER>`: on Forgejo 17 and later, and GitHub
  - `get`, `close`, `reopen`, `delete` and `lock` refuse a pull request's number, which the server would take for an issue's, even to delete it; `edit` and `comment` accept it
  - `issue comment list|add|edit|delete` (alias `comments`): an issue's comments
  - `issue label list|get|create|edit|delete` (alias `labels`): a repository's labels
  - `issue milestone list|get|create|edit|delete` (aliases `milestones`, `ms`): a repository's milestones, by title or ID
- `webhook` commands (aliases `webhooks`, `hook`, `hooks`), for five kinds of webhooks: `webhook user` (your own), `webhook system` (the whole server), `webhook default` (templates copied into new repositories), all three on Forgejo, and `webhook repo` and `webhook org`, on Forgejo, GitHub and Bitbucket. See "Webhooks" in the README:
  - `list [-j]` (alias `ls`): lists the webhooks; `-q` outputs only their IDs. Not on `default`, which Forgejo's API cannot list
  - `get [-j] <ID>` (alias `info`): shows a webhook, with all of its events; authorization headers are never shown
  - `create [flags] <URL>` (alias `add`): creates a webhook, subscribed to push events unless `-e`, `--event` says otherwise (`all`: everything); `-q` outputs only its ID. Secrets and authorization headers are prompted for or read from a file, never given on the command line
  - `edit [flags] <ID>` (aliases `modify`, `update`): only changes what the flags passed say. Refuses what Forgejo's API would silently ignore: changing `package`, `action_run_*` and `workflow_*` events, changing the secret, removing the authorization header
  - `delete [-y] <ID>...` (aliases `rm`, `remove`): deletes webhooks, after confirmation
  - `ping <ID>` (alias `test`): sends a test delivery, on `repo` and `org`; not on Bitbucket, and on Forgejo, repositories only
- From inside a clone (anywhere in it), `branch`, `branch protection`, `tag`, `release`, `pr`, `issue`, `history`, `repo info`, `repo edit` and `webhook repo` work out the repository and its owner, so `<REPO>` and `-o` can be left out; `-c` picks which forge's remote is read. See "Working from inside a clone" in the README:
  - with several targets, the first argument remains the repository; `.` stands for the current clone's: `tag rm . v1 v2`. Pull request numbers need no `.`: `pr close 12 13`
  - `repo delete` still needs the name
  - each command's `--help` tells what applies to it

### Changed

- A repository named on the command line, without `-o`, now takes its owner from the current clone when it is that clone's repository; any other keeps the usual owner
- `branch list` no longer refuses to run without a repository: it reads the current clone's
- `pr close`, `pr reopen` and `pr reviewers` no longer take a number as the repository, so a repository with a numeric name cannot be designated by them; neither can the new `webhook repo delete`
- The confirmation questions of `branch delete` (server side), `tag delete`, `release delete` and `release delete-assets` now name the repository
- Built on gitoperationslib 1.5.1, for its organization avatars and webhook APIs. **Forgejo servers must now run 15.0.0 or later** (it was 11.0.10)
- `user whoami` now always shows the 2FA status, which used to need `-x`; it reads `not reported` when the forge does not tell, as Forgejo's API never does

## [1.3.0] - 2026-11-04

### Added

- `branch protection` commands (aliases `protections`, `protect`, `prot`): a repository's branch protection rules, addressed by their pattern, on Forgejo, GitHub (branch rulesets) and Bitbucket (branch restrictions):
  - `branch protection list [-o OWNER] [-j] <REPO>` (alias `ls`): lists the rules; `-q` outputs only their patterns
  - `branch protection get [-o OWNER] [-j] <REPO> <PATTERN>` (alias `info`): shows a rule, with the settings this forge has
  - `branch protection create [-o OWNER] [flags] <REPO> <PATTERN>` (alias `add`): creates a rule; without any flag, it still blocks deleting the matching branches and force-pushing to them
  - `branch protection edit [-o OWNER] [flags] <REPO> <PATTERN>` (alias `update`): only changes the settings whose flag is passed. On GitHub, the whole ruleset is written back: its name, its enforcement status and any setting gitclt does not know of are overwritten
  - `branch protection delete [-o OWNER] [-y] <REPO> <PATTERN>...` (aliases `rm`, `del`): deletes rules, after confirmation
- Two-factor authentication on Forgejo password logins, for every command: a password-only connection on such an account used to fail outright (tokens need no code):
  - `--otp CODE` is now a global flag; it was only on `token create` and `token delete`
  - without it, the code is asked for when the server needs one and there is a terminal (three attempts)
  - a new code is asked for when a long run outlasts the current one

### Changed

- `repo delete` now asks for confirmation, and checks that the repository exists first; `-y`, `--yes` skips the question. **Scripts must now pass `-y`**: without a terminal to ask on, nothing is deleted

### Fixed

- `repo create` failed on GitHub and Bitbucket unless `--default-branch ''` was passed, as its default of `develop` was refused there: `develop` is now only asked for where the forge can set it
- `repo list -q` printed the whole table, instead of the `owner/name` of each repository
- A user that Bitbucket cannot find among a workspace's members (`branch protection`, `pr` reviewers) now gets told which names it accepts, and that `user whoami -x` shows them

## [1.2.0] - 2026-11-04

### Added

- `token` commands (alias `tokens`): the access tokens of the connection's user, on Forgejo (GitHub and Bitbucket have no API for this):
  - `token list [-j]` (alias `ls`): lists the tokens; `-q` outputs only their names
  - `token info [-j] <NAME>` or `--id ID` (alias `get`): shows a token
  - `token create -s SCOPE [-r OWNER/REPO] [--otp CODE] [-j] <NAME>` (alias `add`): creates a token, and shows its secret, once; `-q` outputs only the secret
  - `token delete [--otp CODE] [-y] <NAME>` or `--id ID` (aliases `rm`, `remove`): deletes a token, after confirmation; warns if it looks like the connection's own token
  - `create` and `delete` authenticate with the connection's password, as Forgejo refuses tokens there; it is asked for if the connection does not store one

## [1.1.0] - 2026-10-30

### Added

- `history` commands (aliases `log`, `commits`): a repository's commits, read from the server like `git log` (no local clone needed; only pushed commits are seen), on Forgejo, GitHub and Bitbucket:
  - `history list [-o OWNER] [-r REF] [-p PATH] [-n COUNT] [--since DATE] [--until DATE] [-j] <REPO>` (alias `ls`): lists the commits reachable from a branch, tag or commit (default: the default branch), newest first, with their date, author and title; merge commits are marked with `M`. `-p` keeps the commits that touched a file or directory, `-n` caps the count (default 30, `0`: the whole history). `--since` / `--until` (a day or an RFC 3339 timestamp) are GitHub only
  - `history get [-o OWNER] [-j] <REPO> <REF>` (aliases `show`, `info`): shows a commit (SHA, branch or tag), like `git show --stat`: author, committer, parents, line totals, full message and changed files (no per-file line counts nor rename detection on Forgejo)

### Changed

- Built on gitoperationslib 1.1.0, for its commit history API

<br><br>
## [1.0.0] - 2026-10-29

First production-ready release.

### Added

- `pr` commands (aliases `pull`, `pullrequest`), on Forgejo, GitHub and Bitbucket:
  - `pr list [-o OWNER] [-s open|closed|all] [-j] <REPO>` (alias `ls`): lists the pull requests, open ones by default, with their branches, author and status (open, draft, closed, merged)
  - `pr get [-o OWNER] [-j] <REPO> <NUMBER>` (alias `info`): shows a pull request, with its mergeability, reviews and description
  - `pr create [-o OWNER] [-t TITLE] [-b BODY | -F FILE] [-d] [-r USER,...] <REPO> <HEAD> [<BASE>]` (aliases `add`, `open`): opens a pull request into `<BASE>` (default: the default branch), as a draft with `-d`, and requests reviews with `-r`; both branches are checked first
  - `pr edit [-o OWNER] [flags] <REPO> <NUMBER>`: only the flags passed are sent (`--title`, `--body`/`--body-file`, `--base`, `--draft`)
  - `pr merge [-o OWNER] [-m METHOD] [-d] [--auto | --cancel-auto] <REPO> <NUMBER>`: merges with a method (merge, squash, rebase...), deleting the head branch with `-d`; refuses drafts, closed and conflicting pull requests, and a head that moved since it was checked. `--auto` merges once the checks pass
  - `pr close` / `pr reopen [-o OWNER] <REPO> <NUMBER>...`: every number is checked first; those already in the wanted state are skipped, and a merged pull request cannot be reopened (Bitbucket cannot reopen at all)
  - `pr review [-o OWNER] (-a | -x | -n) [-m TEXT] <REPO> <NUMBER>`: approves, requests changes or comments, as the connection's user
  - `pr reviewers [-o OWNER] [--remove] <REPO> <NUMBER> <USER>...`: requests (or withdraws) reviews

### Changed

- Built on gitoperationslib 1.0.0 (its first stable release): a merged pull request whose head branch was deleted now shows that branch's name, not `refs/pull/N/head`

<br><br>
## [0.50.0] - 2026-10-29

### Added

- `release edit [-o OWNER] [flags] <REPO> <TAG>` (aliases `modify`, `update`): edits the release of a tag, drafts included; only the flags passed are sent: `--title`, `--notes` or `--notes-file FILE`, `--draft`, `--prerelease`. Booleans are revoked with `=false`, and an empty string clears a text field
- `release publish [-o OWNER] <REPO> <TAG>...` (alias `rel pub`): publishes the draft releases of one or more tags, skipping those already published. Every tag is checked first (it must have a release, and be on the server); the run stops at the first failure. `release edit --draft=false` publishes too, with the same tag check. Both are Forgejo and GitHub only
- `tag` commands, remote-only: they work on the server's tags (Forgejo, GitHub and Bitbucket), never on the local ones, which are left to git:
  - `tag list [-o OWNER] [-j] <REPO>` (alias `ls`): lists the tags, newest version first, with their commit, release (Forgejo, GitHub) and annotation (not in GitHub's tag list)
  - `tag get [-o OWNER] [-j] <REPO> <TAG>` (alias `info`): shows a tag's commit, type (annotated or lightweight), annotation, release and source archive URLs
  - `tag create [-o OWNER] [-t TARGET] [-m MESSAGE] <REPO> <TAG>` (alias `add`): creates a tag on a branch, tag or commit (default: the default branch); `-m` makes it annotated. An existing tag is refused
  - `tag delete [-o OWNER] [--with-release] [-y] <REPO> <TAG>...` (aliases `rm`, `del`): deletes tags; every tag is checked before the confirmation, and a tag that still has a release is refused, unless `--with-release` deletes the release (and its assets) too. The run stops at the first failure

### Changed

- `release assets` is renamed `release list-assets` (alias `rel lsa` unchanged), and `release delete-assets` gets a second alias, `rel dla`

<br><br>
## [0.30.0] - 2026-10-29

### Added

- `release list [-o OWNER] [-j] <REPO>` (alias `rel ls`): lists the releases of a repository, drafts included, with their title, type (release, prerelease, draft), publication date, author and number of assets
- `release assets [-o OWNER] [-j] <REPO> <TAG>` (alias `rel lsa`): lists the files attached to the release of a tag, with their size, download count and upload date; a draft's release is found too
- `release delete [-o OWNER] [-y] <REPO> <TAG>...` (alias `rel rm`): deletes the releases of one or more tags, drafts included, with their assets; the tags themselves stay on the server
- `release delete-assets [-o OWNER] [-y] <REPO> <TAG> <ASSET>...` (alias `rel rma`): deletes one or more files from the release of a tag
- Both deletions check every tag (or asset name) before asking for confirmation (`-y`, `--yes` skips it), and report all the missing ones at once; the run stops at the first failed deletion, and what was deleted before it stays deleted. All four commands are Forgejo and GitHub only

<br><br>
## [0.21.0] - 2026-10-29

### Added

- `release upload [-o OWNER] [--create [--draft] [--prerelease]] [-r] <REPO> <TAG> <FILE>...` (aliases `rel up`): uploads files (ex: the RPM / DEB / APK packages a CI job builds) to the release of a tag, each under its base name. `--create` creates a missing release, titled after the tag; the tag must already be on the server. `-r`, `--replace` overwrites the assets that already have the same name; without it, a name already taken is an error. Everything is checked before the first upload, and a re-run of `--create --draft` finds the draft it created instead of creating another one. Forgejo and GitHub only

### Changed

- `release upload` checks first that the repository exists: when `<REPO>` or `<TAG>` is left out, the files shift into their place, and this is now reported as `No such repository`, with the expected usage, instead of a 404 on the release list
- `release upload --help` explains its arguments, has more examples, and its flags are described in more detail
- Every command's usage line now shows its arguments (ex: `gitclt release upload <REPO> <TAG> <FILE>... [flags]`, `gitclt branch list [<REPO>] [flags]`); they used to show `[flags]` only. In `branch rename`, `<NEW NAME>` is now spelled `<NEW-NAME>`, like in `org rename`

<br><br>
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
