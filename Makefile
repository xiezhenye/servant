pwd=$(shell pwd)
arch=$(shell echo `go env GOOS`_`go env GOARCH`)
drivers_file=src/servant/server/sql_drivers.go

.PHONY : all clean driver tarball

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

bin/servant:$(drivers_file) src/servant.go
	GOPATH=$(pwd) GOBIN=$(pwd)/bin go install src/servant.go

tarball:servant.tar.gz
	mkdir servant
	cp -r bin conf README.md servant
	tar -czf servant.tar.gz servant
	rm -rf servant
	
servant.tar.gz:bin/servant

clean:
	rm -rf servant bin pkg/$(arch)/servant "$(drivers_file)" servant.tar.gz


