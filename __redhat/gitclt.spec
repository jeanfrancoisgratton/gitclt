%define debug_package   %{nil}
%define _build_id_links none
%define _name gitclt
%define _prefix /opt
%define _version 1.0.0
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
#BuildArchitectures: x86_64
BuildRequires: gcc
#Requires: sudo
#Obsoletes: vmman1 > 1.140

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
