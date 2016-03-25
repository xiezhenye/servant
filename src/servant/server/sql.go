package server

import (
	"database/sql"
	"regexp"
	"servant/conf"
	"net/http"
	"encoding/json"
)

//var paramRe, _ = regexp.Compile(`^\$\w+$`)
var dbUrlRe, _ = regexp.Compile(`^/database/(\w+)/(\w+)/?$`)

type DatabaseServer struct {
	*Session
}

func NewDatabaseServer(sess *Session) Handler {
	return &DatabaseServer{
		Session:sess,
	}
}

func (self *DatabaseServer) findDatabaseQueryConfigByPath(path string) (*conf.Database, *conf.Query) {
	m := dbUrlRe.FindStringSubmatch(path)
	if len(m) != 3 {
		return nil, nil
	}
	dbConf, ok := self.config.Databases[m[1]]
	if !ok {
		return nil, nil
	}
	qConf, ok := dbConf.Queries[m[2]]
	if !ok {
		return dbConf, nil
	}
	return dbConf, qConf
}

func (self *DatabaseServer) serve() {
	urlPath := self.req.URL.Path
	method := self.req.Method
	self.info("database", "+ %s %s %s", self.req.RemoteAddr, method, urlPath)
	_, err := self.auth()
	if err != nil {
		self.warn("database", "- auth failed: %s", err.Error())
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}

	if method != "GET" {
		self.warn("database", "- not allow method: %s", method)
		self.resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	dbConf, queryConf := self.findDatabaseQueryConfigByPath(urlPath)
	if dbConf == nil {
		self.warn("database", "- database %s not found", urlPath)
		self.resp.WriteHeader(http.StatusNotFound)
		return
	}
	if queryConf == nil {
		self.warn("database", "- query %s not found", urlPath)
		self.resp.WriteHeader(http.StatusNotFound)
		return
	}

	db, err := sql.Open(dbConf.Driver, dbConf.Dsn)
	if err != nil {
		self.warn("database", "- driver init failed")
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer db.Close()

	data, err := dbQuery(db, queryConf.Sql)
	if err != nil {
		self.warn("database", "- query %s failed: %s", queryConf.Sql, err)
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf, err := json.Marshal(data)
	if err != nil {
		self.warn("database", "- json marshal failed: %s", err)
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	self.resp.Write(buf)
}

func dbQuery(db *sql.DB, sql string) ([]map[string]string, error) {
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	ret := make([]map[string]string, 0, 1)
	row := make([]interface{}, len(columns))
	for i, _ := range(row) {
		row[i] = new(string)
	}
	for rows.Next() {
		err = rows.Scan(row...)
		if err != nil {
			return ret, err
		}
		mapRow := make(map[string]string)
		for i, column := range(columns) {
			mapRow[column] = *(row[i]).(*string)
		}
		ret = append(ret, mapRow)
	}
	return ret, nil
}