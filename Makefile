pwd=$(shell pwd)
arch=$(shell echo `go env GOOS`_`go env GOARCH`)
drivers_file=src/servant/server/sql_drivers.go

.PHONY : all clean driver tarball test

all:bin/servant

DRIVERS="mysql"

driver:$(drivers_file)

$(drivers_file):
	[ -e "$(drivers_file)" ] || ( echo 'package server'; \
	echo 'import (' ; \
	for d in $(DRIVERS); do \
		case "$$d" in \
		mysql) \
			GOPATH=$(pwd) go get github.com/go-sql-driver/mysql; \
			echo '_ "github.com/go-sql-driver/mysql"' ;; \
		sqlite) \
			GOPATH=$(pwd) go get github.com/mattn/go-sqlite3;  \
			echo '_ "github.com/mattn/go-sqlite3"' ;; \
		postgresql) \
			GOPATH=$(pwd) go get github.com/lib/pq;  \
			echo '_ "github.com/lib/pq"' ;; \
		esac \
	done ; \
	echo ')' ) >"$(drivers_file)"



bin/servant:$(arch)/bin/servant
	cp -r $(arch)/bin .

linux_amd64/bin/servant:driver
	GOOS=linux GOARCH=amd64 GOPATH=$(pwd) CGO_ENABLED=1 GOBIN=$(pwd)/linux_amd64/bin go install -v src/servant.go

darwin_amd64/bin/servant:driver
	GOOS=darwin GOARCH=amd64 GOPATH=$(pwd) CGO_ENABLED=1 GOBIN=$(pwd)/darwin_amd64/bin go install -v src/servant.go


tarball:servant.tar.gz

servant.tar.gz:bin/servant
	mkdir servant
	cp -r bin conf README.md LICENSE servant
	tar -czf servant.tar.gz servant
	rm -rf servant

servant-src.tar.gz:driver
	mkdir servant-src
	cp -r src conf example README.md Makefile scripts LICENSE servant-src
	find servant-src -name '.git*' | xargs rm -rf
	tar -czvf servant-src.tar.gz servant-src
	rm -rf servant-src

rpm:servant-src.tar.gz
	mkdir -p rpmbuild/{SPECS,SOURCES}
	cp servant-src.tar.gz rpmbuild/SOURCES
	cp servant.spec rpmbuild/SPECS
	rpmbuild --define "_topdir $(pwd)/rpmbuild" -ba rpmbuild/SPECS/servant.spec
	mv rpmbuild/SRPMS/*.src.rpm .
	mv rpmbuild/RPMS/x86_64/*.rpm .
	rm -rf rpmbuild


test:
	GOPATH=$(pwd) go test -v -coverprofile=c_server.out servant/server
	GOPATH=$(pwd) go test -v -coverprofile=c_conf.out servant/conf

clean:
	rm -rf servant bin pkg/*/servant "$(drivers_file)" servant.tar.gz servant-src.tar.gz darwin_amd64 linux_amd64 rpmbuild servant-src c_server.out c_conf.out *.rpm pkg


