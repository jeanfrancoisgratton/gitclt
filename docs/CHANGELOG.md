# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
