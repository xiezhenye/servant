package server

import (
	"net/http"
	"regexp"
	"servant/conf"
	"path"
	//"fmt"
	"strings"
	"os"
	"io"
)

var fileUrlRe, _ = regexp.Compile(`^/files/(\w+)/(\w+)(/.+)$`)


func (self *Session) findDirConfigByPath(path string) (*conf.Dir, string) {
	m := fileUrlRe.FindStringSubmatch(path)
	if len(m) != 4 {
		return nil, ""
	}
	filesConf, ok := self.config.Files[m[1]]
	if !ok {
		return nil, ""
	}
	dirConf, ok := filesConf.Dirs[m[2]]
	if !ok {
		return nil, ""
	}
	return dirConf, m[3]
}

func (self *Session) checkDirAllow(dirConf *conf.Dir, method string) error {
	return nil
}

func (self *Session) serveFile(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	dirConf, relPath := self.findDirConfigByPath(req.URL.Path)
	if dirConf == nil {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	if self.checkDirAllow(dirConf, req.Method) != nil {
		resp.WriteHeader(http.StatusForbidden)
		return
	}
    filePath := path.Clean(dirConf.Root + relPath)
	if ! strings.HasPrefix(filePath, path.Clean(dirConf.Root) + "/") {
		resp.WriteHeader(http.StatusForbidden)
		return
	}
	var file *os.File
	defer file.Close()
	var err error
	switch req.Method {
	case "GET":
		file, err = os.Open(filePath)
		if err != nil {
			resp.Header().Set("X-SERVANT-ERR", err.Error())
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.Copy(resp, file)
	case "POST":
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0664)
		if err != nil {
			resp.Header().Set("X-SERVANT-ERR", err.Error())
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.Copy(file, req.Body)
	case "DELETE":
		err = os.Remove(filePath)
		if err != nil {
			resp.Header().Set("X-SERVANT-ERR", err.Error())
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "PUT":
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0664)
		if err != nil {
			resp.Header().Set("X-SERVANT-ERR", err.Error())
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.Copy(file, req.Body)
	}
	//fmt.Println(filePath)
}
