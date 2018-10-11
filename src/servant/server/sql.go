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
	reqParams := requestParams(self.req)
	if !ValidateParams(queryConf.Validators, reqParams) {
		self.ErrorEnd(http.StatusBadRequest, "validate params failed")
		return
	}
	db, err := sql.Open(dbConf.Driver, dbConf.Dsn)
	if err != nil {
		self.ErrorEnd(http.StatusInternalServerError, "driver init failed")
		return
	}
	defer db.Close()
	data := make([]sqlResult, 0, 1)

	for _, sql := range(queryConf.Sqls) {
		sqlReplaced, sqlParams, ok := replaceSqlParams(sql, reqParams)
		if !ok {
			self.ErrorEnd(http.StatusInternalServerError, "parse sql params failed. sql: %s, params: %v", sql, self.req.URL.Query())
			return
		}
		result, err := dbQuery(db, sqlReplaced, sqlParams)
		if err != nil {
			self.ErrorEnd(http.StatusInternalServerError, "query %s failed: %s", sqlReplaced, err)
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

func replaceSqlParams(inSql string, query ParamFunc) (string, []interface{}, bool){
	params := make([]interface{}, 0, 4)
	outSql, ok := VarExpand(inSql, query, func(s string)string {
		params = append(params, s)
		return "?"
	})
	return outSql, params, ok
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

