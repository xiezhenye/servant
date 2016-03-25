package server

import (
	"database/sql"
	"servant/conf"
	"net/http"
	"encoding/json"
)

//var paramRe, _ = regexp.Compile(`^\$\w+$`)

type DatabaseServer struct {
	*Session
}

func NewDatabaseServer(sess *Session) Handler {
	return &DatabaseServer{
		Session:sess,
	}
}

func (self *DatabaseServer) findDatabaseQueryConfigByPath(path string) (*conf.Database, *conf.Query) {
	dbConf, ok := self.config.Databases[self.group]
	if !ok {
		return nil, nil
	}
	qConf, ok := dbConf.Queries[self.item]
	if !ok {
		return dbConf, nil
	}
	return dbConf, qConf
}

func (self *DatabaseServer) serve() {
	urlPath := self.req.URL.Path
	method := self.req.Method

	if method != "GET" {
		self.ErrorEnd(http.StatusMethodNotAllowed, "not allow method: %s", method)
		return
	}
	dbConf, queryConf := self.findDatabaseQueryConfigByPath(urlPath)
	if dbConf == nil {
		self.ErrorEnd(http.StatusNotFound, "database %s not found", urlPath)
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