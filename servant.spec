
%define _target_os Linux
%define oname servant
%define _prefix /opt/%{oname}
%define _docdir %{_prefix}/doc

Summary: A common agent to execute commands, serve files and so on
Name: servant
Version: 0.16
Release: 4%{?dist}
License: APL
Group: Applications/System
URL: http://github.com/xiezhenye/servant

Source0: %{oname}-src.tar.gz
#BuildRequires: yz-go

%description
A common agent to execute commands, serve files and so on


%prep
%setup -n %{oname}-src

%build
%define debug_package %{nil}
%{__make} linux_amd64/bin/servant

%install
%{__rm} -rf %{buildroot}
%{__mkdir} -p %{buildroot}%{_prefix}/doc
%{__mv} linux_amd64/bin %{buildroot}%{_prefix}
%{__mv} README.md LICENSE example %{buildroot}%{_docdir}
%{__mv} conf %{buildroot}%{_prefix}
%{__mv} scripts %{buildroot}%{_prefix}

%clean
%{__rm} -rf %{buildroot}

%files
%defattr(-, root, root, 0755)
%{_bindir}/servant
%{_prefix}/scripts/servantctl
%{_prefix}/conf/extra
%defattr(-, root, root, 0644)
%{_docdir}/README.md
%{_docdir}/LICENSE
%{_docdir}/example/example.xml
%{_docdir}/example/timer.xml
%{_prefix}/conf/servant.xml

