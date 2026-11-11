## Post-1.0.0

### User self-service commands (non-admin, acting on the authenticated user)
- [x] ~~GPG keys: list / add / remove / verify (`/user/gpg_keys`, `/user/gpg_key_verify`)~~
- [ ] Profile edit for oneself (`PATCH /user/settings`; GitHub `PATCH /user`)

### Repository features and settings (gitoperationslib 1.7.0)
Found 2026-11-10: Actions cannot be enabled on a repository from gitclt, which blocks stubber's Forgejo Actions validation.
- [x] ~~`repo create` with Actions on by default, `--actions=false` to prevent it~~
- [x] ~~`repo edit` flags for the features (Actions, issues, wiki, pull requests, projects, releases, packages, discussions) and the merge, issue tracker, wiki and Actions settings~~
- [x] ~~Show the features and settings in `repo info` (table and `--json`)~~

<br><br><br>
