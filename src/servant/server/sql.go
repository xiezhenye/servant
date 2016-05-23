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

type sqlResult []map[string]string

func NewDatabaseServer(sess *Session) Handler {
	return DatabaseServer{
		Session:sess,
	}
}

func (self DatabaseServer) findDatabaseQueryConfig() (*conf.Database, *conf.Query) {
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

func (self DatabaseServer) serve() {
	method := self.req.Method
	if method != "GET" {
		self.ErrorEnd(http.StatusMethodNotAllowed, "not allow method: %s", method)
		return
	}
	dbConf, queryConf := self.findDatabaseQueryConfig()
	if dbConf == nil {
		self.ErrorEnd(http.StatusNotFound, "database not found")
		return
	}
	if queryConf == nil {
		self.ErrorEnd(http.StatusNotFound, "query not found")
		return
	}
	//dsn := replaceCmdParams(dbConf.Dsn, globalParams())
	db, err := sql.Open(dbConf.Driver, dbConf.Dsn)
	if err != nil {
		self.ErrorEnd(http.StatusInternalServerError, "driver init failed")
		return
	}
	defer db.Close()
	data := make([]sqlResult, 0, 1)
	for _, sql := range(queryConf.Sqls) {
		sql, params := replaceSqlParams(sql, requestParams(self.req))

		result, err := dbQuery(db, sql, params)
		if err != nil {
			self.ErrorEnd(http.StatusInternalServerError, "query %s failed: %s", sql, err)
			return
		}
		data = append(data, result)
	}
	buf, err := json.Marshal(data)
	if err != nil {
		self.ErrorEnd(http.StatusInternalServerError, "json marshal failed: %s", err)
		return
	}
	self.resp.Write(buf)
	self.GoodEnd("execution done")
}

func replaceSqlParams(inSql string, query func(string)string) (outSql string, params []interface{}){
	params = make([]interface{}, 0, 4)
	return paramRe.ReplaceAllStringFunc(inSql, func(s string) string {
		var v string = ""
		if query != nil {
			v = query(s[2:len(s) - 1])
		}
		params = append(params, v)
		return "?"
	}), params
}

func dbQuery(db *sql.DB, sql string, params []interface{}) (sqlResult, error) {
	rows, err := db.Query(sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToResult(rows)
}

func rowsToResult(rows *sql.Rows) (sqlResult, error) {
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

