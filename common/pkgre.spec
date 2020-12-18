################################################################################

# rpmbuilder:relative-pack true

################################################################################

%define  debug_package %{nil}

################################################################################

%define _posixroot        /
%define _root             /root
%define _bin              /bin
%define _sbin             /sbin
%define _srv              /srv
%define _home             /home
%define _opt              /opt
%define _lib32            %{_posixroot}lib
%define _lib64            %{_posixroot}lib64
%define _libdir32         %{_prefix}%{_lib32}
%define _libdir64         %{_prefix}%{_lib64}
%define _logdir           %{_localstatedir}/log
%define _rundir           %{_localstatedir}/run
%define _lockdir          %{_localstatedir}/lock/subsys
%define _cachedir         %{_localstatedir}/cache
%define _spooldir         %{_localstatedir}/spool
%define _crondir          %{_sysconfdir}/cron.d
%define _loc_prefix       %{_prefix}/local
%define _loc_exec_prefix  %{_loc_prefix}
%define _loc_bindir       %{_loc_exec_prefix}/bin
%define _loc_libdir       %{_loc_exec_prefix}/%{_lib}
%define _loc_libdir32     %{_loc_exec_prefix}/%{_lib32}
%define _loc_libdir64     %{_loc_exec_prefix}/%{_lib64}
%define _loc_libexecdir   %{_loc_exec_prefix}/libexec
%define _loc_sbindir      %{_loc_exec_prefix}/sbin
%define _loc_bindir       %{_loc_exec_prefix}/bin
%define _loc_datarootdir  %{_loc_prefix}/share
%define _loc_includedir   %{_loc_prefix}/include
%define _loc_mandir       %{_loc_datarootdir}/man
%define _rpmstatedir      %{_sharedstatedir}/rpm-state
%define _pkgconfigdir     %{_libdir}/pkgconfig

%define __service         %{_sbin}/service
%define __chkconfig       %{_sbin}/chkconfig
%define __sysctl          %{_bindir}/systemctl

################################################################################

%define morpher_user      morpher
%define morpher_group     morpher
%define src_dir           src/github.com/essentialkaos/%{name}

################################################################################

Summary:            pkg.re morpher server
Name:               pkgre
Version:            4.2.1
Release:            0%{?dist}
Group:              Applications/System
License:            Apache License, Version 2.0
URL:                https://kaos.sh/pkgre

Source0:            https://source.kaos.st/pkgre/%{name}-%{version}.tar.bz2

BuildRoot:          %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:      golang >= 1.14

Requires:           systemd

Requires(pre):      shadow-utils
Requires(post):     systemd
Requires(preun):    systemd
Requires(postun):   systemd

Provides:           %{name} = %{version}-%{release}

################################################################################

%description
pkg.re service morpher server.

################################################################################

%prep
%setup -q

%build
export GOPATH=$(pwd)

pushd src/github.com/essentialkaos/%{name}
  %{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -dm 755 %{buildroot}%{_sysconfdir}
install -dm 755 %{buildroot}%{_sysconfdir}/cron.d
install -dm 755 %{buildroot}%{_sysconfdir}/logrotate.d
install -dm 755 %{buildroot}%{_logdir}
install -dm 755 %{buildroot}%{_logdir}/%{name}/morpher

install -pm 755 %{src_dir}/morpher-server \
                %{buildroot}%{_bindir}/

install -pm 644 %{src_dir}/common/morpher.knf \
                %{buildroot}%{_sysconfdir}/

install -dm 755 %{buildroot}%{_unitdir}
install -pm 644 %{src_dir}/common/morpher.service \
                %{buildroot}%{_unitdir}/

install -pm 755 %{src_dir}/common/morpher.logrotate \
                %{buildroot}%{_sysconfdir}/logrotate.d/morpher

%pre
getent group %{morpher_group} >/dev/null || groupadd -r %{morpher_group}
getent passwd %{morpher_user} >/dev/null || useradd -r -M -g %{morpher_group} -s /sbin/nologin %{morpher_user}
exit 0

%post
if [[ $1 -eq 1 ]] ; then
  %{__sysctl} enable morpher.service &>/dev/null || :
fi

%preun
if [[ $1 -eq 0 ]] ; then
  %{__sysctl} --no-reload disable morpher.service &>/dev/null || :
  %{__sysctl} stop morpher.service &>/dev/null || :
fi

%postun
if [[ $1 -ge 1 ]] ; then
  %{__sysctl} daemon-reload &>/dev/null || :
fi

%clean
rm -rf %{buildroot}

################################################################################

%files
%defattr(-,root,root,-)
%doc LICENSE
%attr(-,%{morpher_user},%{morpher_group}) %dir %{_logdir}/%{name}/morpher/
%config(noreplace) %{_sysconfdir}/morpher.knf
%config(noreplace) %{_sysconfdir}/logrotate.d/morpher
%{_bindir}/morpher-server
%{_unitdir}/morpher.service

################################################################################

%changelog
* Fri Dec 18 2020 Anton Novojilov <andy@essentialkaos.com> - 4.2.1-0
- Improved URL generation for pkg.go.dev redirect

* Fri Dec 18 2020 Anton Novojilov <andy@essentialkaos.com> - 4.2.0-0
- Added redirect to pkg.go.dev

* Tue Dec 08 2020 Anton Novojilov <andy@essentialkaos.com> - 4.1.0-0
- Fixed bug with proxying requests to GitHub

* Thu Nov 12 2020 Anton Novojilov <andy@essentialkaos.com> - 4.0.0-0
- Proxying all requests due to problems with Go Modules Services
- ek package updated to v12
- fasthttp package updated to 1.17.0
- Code refactoring

* Thu Dec 05 2019 Anton Novojilov <andy@essentialkaos.com> - 3.7.3-0
- ek updated to v11

* Mon Jul 22 2019 Anton Novojilov <andy@essentialkaos.com> - 3.7.2-0
- Removed Librato support

* Thu Feb 21 2019 Anton Novojilov <andy@essentialkaos.com> - 3.7.1-0
- Fixed major bug with refs rewriting

* Tue Feb 05 2019 Anton Novojilov <andy@essentialkaos.com> - 3.7.0-0
- ek package updated to v10
- librato package updated to v8
- fasthttp package replaced by original package
- Code refactoring

* Wed Mar 28 2018 Anton Novojilov <andy@essentialkaos.com> - 3.6.0-0
- fasthttp package replaced by erikdubbelboer fork
- Added files limit to init script and systmed unit
- Added systemd unit
- Added autostart

* Tue Mar 06 2018 Anton Novojilov <andy@essentialkaos.com> - 3.5.0-1
- Rebuilt with Go 1.10
- ek package updated to latest release
- fasthttp package updated to latest release

* Tue Oct 31 2017 Anton Novojilov <andy@essentialkaos.com> - 3.5.0-0
- Proxying request's from GoDoc to GitHub

* Fri Oct 13 2017 Anton Novojilov <andy@essentialkaos.com> - 3.4.0-1
- Improved init script
- Improved spec

* Sun May 21 2017 Anton Novojilov <andy@essentialkaos.com> - 3.4.0-0
- ek package updated to v9
- librato package updated to v7

* Sun Apr 16 2017 Anton Novojilov <andy@essentialkaos.com> - 3.3.0-0
- Morpher now return 404 if can't find proper tag/branch
- Code refactoring
- ek package updated to v8
- librato package updated to v6

* Tue Apr 11 2017 Anton Novojilov <andy@essentialkaos.com> - 3.2.0-0
- Return original symref if target version is tag

* Sun Apr 09 2017 Anton Novojilov <andy@essentialkaos.com> - 3.1.0-0
- Default HTTP client replaced by fasthttp client

* Tue Mar 28 2017 Anton Novojilov <andy@essentialkaos.com> - 3.0.0-0
- ek package updated to v7
- librato package updated to v5
- Improved Makefile

* Wed Feb 22 2017 Anton Novojilov <andy@essentialkaos.com> - 2.1.0-0
- ek package updated to v6
- librato package updated to v4

* Thu Oct 27 2016 Anton Novojilov <andy@essentialkaos.com> - 2.0.0-0
- ek package updated to v5
- Removed Librato monitoring from morpher server
- ek.req replaced by default http client
- net/http HTTP server replaced by valyala/fasthttp
- Added recover to request handler

* Fri Jul 01 2016 Anton Novojilov <andy@essentialkaos.com> - 1.0.2-0
- Fixed compatibility with librato.v2

* Wed Jun 29 2016 Anton Novojilov <andy@essentialkaos.com> - 1.0.1-0
- Fixed bug with updating packages with tags
- Migrated to ek.v2
- Minor improvements

* Fri Mar 18 2016 Anton Novojilov <andy@essentialkaos.com> - 1.0.0-0
- First stable release

* Mon Mar 14 2016 Anton Novojilov <andy@essentialkaos.com> - 0.1.5-0
- Redirect requests from browsers to github

* Mon Mar 14 2016 Anton Novojilov <andy@essentialkaos.com> - 0.1.4-0
- Improved refs processing
- pkg.re usage in source code

* Tue Jan 12 2016 Anton Novojilov <andy@essentialkaos.com> - 0.1.3-0
- Fixed bug for url's without version

* Mon Jan 11 2016 Anton Novojilov <andy@essentialkaos.com> - 0.1.1-0
- Improved error handling

* Sun Nov 02 2014 Anton Novojilov <andy@essentialkaos.com> - 0.1-0
- Initial build
