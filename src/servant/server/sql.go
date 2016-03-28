package server

import (
	"database/sql"
	"servant/conf"
	"net/http"
	"encoding/json"
)

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
	sql, params := replaceSqlParams(queryConf.Sql, self.req.URL.Query())

	data, err := dbQuery(db, sql, params)
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

func replaceSqlParams(inSql string, query map[string][]string) (outSql string, params []interface{}){
	params = make([]interface{}, 0, 4)
	return paramRe.ReplaceAllStringFunc(inSql, func(s string) string {
		v, ok := query[s[2:len(s) - 1]]
		if ok {
			params = append(params, v[0])
 		} else {
			params = append(params, "")
		}
		return "?"
	}), params
}

func dbQuery(db *sql.DB, sql string, params []interface{}) ([]map[string]string, error) {
	rows, err := db.Query(sql, params...)
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