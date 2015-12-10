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

func (self *Session) serveFile() {
	defer self.req.Body.Close()
	method := self.req.Method
	urlPath := self.req.URL.Path
	self.info("file", "+ %s %s %s", self.req.RemoteAddr, method, urlPath)
	dirConf, relPath := self.findDirConfigByPath(urlPath)
	if dirConf == nil {
		self.warn("file", "- dir of %s not found", urlPath)
		self.resp.WriteHeader(http.StatusNotFound)
		return
	}
	if self.checkDirAllow(dirConf, method) != nil {
		self.warn("file", "- dir of %s not allows %s", urlPath, method)
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}
    filePath := path.Clean(dirConf.Root + relPath)
	if ! strings.HasPrefix(filePath, path.Clean(dirConf.Root) + "/") {
		self.warn("file", "- attempt to %s out of root: %s", method, urlPath)
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}
	var file *os.File
	defer file.Close()
	var err error
	switch method {
	case "GET":
		file, err = os.Open(filePath)
		if err != nil {
			self.warn("file", "- open file %s for %s failed", filePath, method)
			self.resp.Header().Set("X-SERVANT-ERR", err.Error())
			self.resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = io.Copy(self.resp, file)
	case "POST":
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0664)
		if err != nil {
			self.warn("file", "- open file %s for %s failed", filePath, method)
			self.resp.Header().Set("X-SERVANT-ERR", err.Error())
			self.resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = io.Copy(file, self.req.Body)
	case "DELETE":
		err = os.Remove(filePath)
		if err != nil {
			self.warn("file", "- open file %s for %s failed", filePath, method)
			self.resp.Header().Set("X-SERVANT-ERR", err.Error())
			self.resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "PUT":
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0664)
		if err != nil {
			self.warn("file", "- open file %s for %s failed", filePath, method)
			self.resp.Header().Set("X-SERVANT-ERR", err.Error())
			self.resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = io.Copy(file, self.req.Body)

	}
	if err != nil {
		self.warn("file", "- io error: %s", err.Error())
	} else {
		self.info("file", "- %s done", method)
	}
}
