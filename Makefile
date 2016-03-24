pwd=$(shell pwd)
arch=$(shell echo `go env GOOS`_`go env GOARCH`)
drivers_file="src/servant/server/sql_drivers.go"

all:bin/servant
.PHONY : all

mysql:
	GOPATH=$(pwd) go get github.com/go-sql-driver/mysql 
	grep -Eq '//mysql' $(drivers_file) || sed -i.old '/\/\/ADD_NEW/i\'$$'\n''_ "github.com\/go-sql-driver\/mysql" \/\/mysql'$$'\n' $(drivers_file)

sqlite:
	GOPATH=$(pwd) go get github.com/mattn/go-sqlite3
	grep -Eq '//sqlite' $(drivers_file) || sed -i.old '/\/\/ADD_NEW/i\'$$'\n''_ "github.com\/mattn\/go-sqlite3" \/\/sqlite'$$'\n' $(drivers_file)

postgresql:
	GOPATH=$(pwd) go get github.com/lib/pq
	grep -Eq '//postgresql' $(drivers_file) || sed -i.old '/\/\/ADD_NEW/i\'$$'\n''_ "github.com\/lib\/pq" \/\/postgresql'$$'\n' $(drivers_file)

bin/servant:
	GOPATH=$(pwd) GOBIN=$(pwd)/bin go install src/servant.go

clean:
	rm -rf servant bin pkg/$(arch)/servant


