pwd=$(shell pwd)
arch=$(shell echo "$(go env GOOS)_$(go env GOARCH)")
drivers_file="src/servant/server/sql_drivers.go"

all:servant
.PHONY : all

mysql:
	GOPATH=$(pwd) go get github.com/go-sql-driver/mysql 
	grep -Eq '//mysql' $(drivers_file) || sed -ri 's/\/\/ADD_NEW/_ github.com\/go-sql-driver\/mysql \/\/mysql\n  \/\/ADD_NEW/' $(drivers_file)

sqlite:
	GOPATH=$(pwd) go get github.com/mattn/go-sqlite3
	grep -Eq '//sqlite' $(drivers_file) || sed -ri 's/\/\/ADD_NEW/_ github.com\/mattn\/go-sqlite3 \/\/sqlite\n  \/\/ADD_NEW/' $(drivers_file)

postgresql:
	GOPATH=$(pwd) go get github.com/lib/pq
	grep -Eq '//postgresql' $(drivers_file) || sed -ri 's/\/\/ADD_NEW/_ github.com\/lib\/pq \/\/postgresql\n  \/\/ADD_NEW/' $(drivers_file)

servant:
	GOPATH=$(pwd) go build src/servant.go

clean:
	rm -rf servant bin pkg


