%define debug_package   %{nil}
%define _build_id_links none
%define _name gitclt
%define _prefix /opt
%define _version 0.21.0
%define _rel 1
%define _arch x86_64
%define _binaryname gitclt

Name:       gitclt
Version:    %{_version}
Release:    %{_rel}
Summary:    forgejo / github management tool

Group:      Repo management
License:    GPL-3.0-or-later
URL:        https://git.famillegratton.net:3000/devops/gitclt.git

Source0:    %{name}-%{_version}.tar.gz
BuildRequires: gcc

%description
forgejo / github management tool

%prep
%autosetup

%build
cd src
go mod download
PATH=$PATH:/opt/go/bin go vet ./...
PATH=$PATH:/opt/go/bin go test ./...
MODPATH=$(PATH=$PATH:/opt/go/bin go list -m)
BUILDDATE=$(date -u +%Y-%m-%d)
PATH=$PATH:/opt/go/bin CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid= -X ${MODPATH}/cmd.buildVersion=%{_version} -X ${MODPATH}/cmd.buildDate=${BUILDDATE}" -o %{_builddir}/%{name}-%{version}/%{_binaryname} .

%clean
rm -rf $RPM_BUILD_ROOT

%pre

%install
rm -rf %{buildroot}
install -Dpm 0755 %{_builddir}/%{name}-%{version}/%{_binaryname} %{buildroot}%{_bindir}/%{_binaryname}

%post

%preun

%postun

%files
%defattr(0755,root,root,-)
%{_bindir}/%{_binaryname}


%changelog
* Wed Oct 28 2026 Binary package builder <builder@famillegratton.net> 0.11.0-1
- Merge remote-tracking branch 'refs/remotes/origin/develop' into develop
- updated changelog
- Merge branch 'feature/branches' into develop
- housekeeping: new release 0.11.0
- feature: branch compare and branch rename
- feature: branch compare and branch rename
- feature: completed branch get
- chore: update changelog for 0.10.0-1

* Tue Oct 27 2026 Binary package builder <builder@famillegratton.net> 0.10.0-1
- version bump
- fixed: branch rm can now remove all local or remote branches
- feature: branch rm
- completed branch ls
- chore: update changelog for 0.9.0-1

* Mon Oct 26 2026 Binary package builder <builder@famillegratton.net> 0.9.0-1
- housekeeping: version bump
- feature: added a new 'wipe' command to fully remove a user
- chore: update changelog for 0.8.2-2
- enhancement: clarifying header titles
- chore: update changelog for 0.8.2-1

* Mon Oct 26 2026 Binary package builder <builder@famillegratton.net> 0.8.2-2
- enhancement: clarifying header titles
- chore: update changelog for 0.8.2-1

* Mon Oct 26 2026 Binary package builder <builder@famillegratton.net> 0.8.2-1
- feature: teams command is complete
- completed teams, and major directory restructure
- Merge branch 'main' into develop
- chore: update changelog for 0.8.0-1

* Sun Oct 25 2026 Binary package builder <builder@famillegratton.net> 0.8.0-1
- version bump
- feature: completed the org subcommand
- bug: removed duplicate field on output
- housekeeping: version bump
- feature: added org edit and org create
- housekeeping: packages renamed in a consistent manner
- chore: completed conn_cfg refactoring
- Merge branch 'feature/orgs_teams' into develop
- refactored conn_cfg
- refactored conn_cfg
- chore: ensuring that we are running a DEV version
- housekeeping: version bump
- enhancement: added full test coverage
- feature: completed org rm
- feature: added the org get command
- org list all completed
- feature: org ls completed
- chore: refactored conn_cfg -> connection
- chore: update changelog for 0.7.0-1

* Sat Oct 24 2026 Binary package builder <builder@famillegratton.net> 0.7.0-1
- housekeeping: version bump
- feature: completed user create and user edit
- feature: new command, user delete
- Merge branch 'develop' into feature/user
- Merge branch 'main' into develop
- Merge remote-tracking branch 'refs/remotes/origin/develop' into develop
- chore: update changelog for 0.6.0-1
- chore: version bump
- Completed user whoami, user get and user ls

* Sat Oct 24 2026 Binary package builder <builder@famillegratton.net> 0.6.0-1
- chore: version bump
- Merge branch 'develop'
- feature: completed user info
- feature: user commands
- stubbing users
- chore: update changelog for 0.5.0-1

* Wed Oct 21 2026 Binary package builder <builder@famillegratton.net> 0.5.0-1
- chore: converted CHANGELOG format to the 'Keep a Changelog' format
- housekeeping: build test on 0.5.0
- feature: completed repo create (all the repo subcommands, for that matter)
- housekeeping: version bump
- housekeeping: version bump
- feature: completed repo rm
- feature: completed repo info
- enhancement : repo ls defaults to -S ID now
- feature: completed repo ls and added bitbucket support
- version bump, removed useless packaging files
- chore: update changelog for 0.1.0-1

* Mon Oct 19 2026 Binary package builder <builder@famillegratton.net> 0.1.0-1
- feature: repo ls is completed
- Merge remote-tracking branch 'refs/remotes/origin/develop' into develop
- First attempt at repo ls
- fixed makefile
- chore (ALPINE): fixed deps
- Don't build package

