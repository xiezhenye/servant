package server

import (
	"net/http"
	"regexp"
	"servant/conf"
	"path"
	"strings"
	"os"
	"io"
	"fmt"
	"errors"
	"strconv"
	"path/filepath"
)

type FileServer struct {
	*Session
}

func NewFileServer(sess *Session) Handler {
	return FileServer{
		Session:sess,
	}
}

func (self FileServer) findDirConfig() (*conf.Dir, string) {
	filesConf, ok := self.config.Files[self.group]
	if !ok {
		return nil, ""
	}
	dirConf, ok := filesConf.Dirs[self.item]
	if !ok {
		return nil, ""
	}
	return dirConf, self.tail
}

func checkDirAllow(dirConf *conf.Dir, relPath string, method string) error {
	ok := false
	for _, allowed := range(dirConf.Allows) {
		if allowed == method {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("method %s not allowed", method)
	}
	if len(dirConf.Patterns) > 0 {
		ok = false
		var err error
		for _, pattern := range(dirConf.Patterns) {
			ok, err = regexp.MatchString(pattern, relPath)
			if err == nil && ok {
				break
			}
		}
		if ! ok {
			return fmt.Errorf("%s not match allowed pattern", relPath)
		}
	}
	return nil
}

func (self FileServer) openFileError(err error, method, filePath  string) {
	e := ""
	if err != nil {
		e = err.Error()
	}
	self.ErrorEnd(http.StatusInternalServerError, "open file %s for %s failed: %s", filePath, method, e)
}

func (self FileServer) serveGet(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		self.openFileError(err, "GET", filePath)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		self.openFileError(err, "GET", filePath)
		return
	}
	rangeStr := self.req.Header.Get("Range")
	ranges, err := parseRange(rangeStr, info.Size())
	if err != nil || len(ranges) > 1 {
		self.ErrorEnd(http.StatusBadRequest, "bad range format or too many ranges(%s) %v", rangeStr, err)
		return
	}
	length := info.Size()
	if ranges != nil && len(ranges) == 1 {
		length = ranges[0].length
		if _, err = file.Seek(ranges[0].start, os.SEEK_SET); err != nil {
			self.openFileError(err, "GET", filePath)
			return
		}
		self.resp.Header().Set("Content-Range", ranges[0].contentRange(info.Size()))
	}
	_, err = io.CopyN(self.resp, file, length)
	if err != nil {
		self.BadEnd("io error: %s", err)
	} else {
		self.GoodEnd("GET done")
	}
}

func (self FileServer) serveHead(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		self.openFileError(err, "HEAD", filePath)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		self.openFileError(err, "HEAD", filePath)
		return
	}
	self.resp.Header().Add("X-Servant-File-Size", strconv.FormatInt(info.Size(), 10))
	self.resp.Header().Add("X-Servant-File-Mtime", info.ModTime().String())
	self.resp.Header().Add("X-Servant-File-Mode", info.Mode().String())
	self.resp.Header().Add("Connection", "close")
	self.GoodEnd("HEAD done")
}

func (self FileServer) servePost(filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0664)
	if err != nil {
		self.openFileError(err, "POST", filePath)
		return
	}
	defer file.Close()
	_, err = io.Copy(file, self.req.Body)
	if err != nil {
		self.ErrorEnd(http.StatusInternalServerError, "io error: %s", err)
	} else {
		self.GoodEnd("POST done")
	}
}

func (self FileServer) servePut(filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		self.openFileError(err, "PUT", filePath)
		return
	}
	defer file.Close()
	_, err = io.Copy(file, self.req.Body)
	if err != nil {
		self.ErrorEnd(http.StatusInternalServerError, "io error: %s", err)
	} else {
		self.GoodEnd("PUT done")
	}
}

func (self FileServer) serveDelete(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		self.ErrorEnd(http.StatusInternalServerError, "delete file fail: %s", err)
	} else {
		self.GoodEnd("DELETE done")
	}
}

func (self FileServer) serveUnknown(filePath string) {
	self.ErrorEnd(http.StatusMethodNotAllowed, "method not allowd")
}

func (self FileServer) serve() {
	method := self.req.Method
	urlPath := self.req.URL.Path

	dirConf, relPath := self.findDirConfig()
	if dirConf == nil {
		self.ErrorEnd(http.StatusNotFound, "dir of %s not found", urlPath)
		return
	}
	err := checkDirAllow(dirConf, relPath, method)
	if err != nil {
		self.ErrorEnd(http.StatusForbidden, err.Error())
		return
	}
	params := requestParams(self.req)
	if !ValidateParams(dirConf.Validators, params) {
		self.ErrorEnd(http.StatusBadRequest, "validate params failed")
		return
	}
	rootDir := replaceCmdParams(dirConf.Root, params)
	filePath := path.Clean(filepath.Join(rootDir, relPath))
	if ! strings.HasPrefix(filePath, path.Clean(rootDir) + "/") {
		self.ErrorEnd(http.StatusForbidden, "attempt to %s out of root: %s", method, relPath)
		return
	}
	self.funcByMethod(method)(filePath)
}

func (self FileServer) funcByMethod(method string) func(string) {
	switch method {
	case "HEAD":
		return self.serveHead
	case "GET":
		return self.serveGet
	case "POST":
		return self.servePost
	case "DELETE":
		return self.serveDelete
	case "PUT":
		return self.servePut
	}
	return self.serveUnknown
}

// copied from go source
// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}
// parseRange parses a Range header string as per RFC 2616.
func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	var badRange = errors.New("invalid range")
	if !strings.HasPrefix(s, b) {
		return nil, badRange
	}
	ranges := make([]httpRange, 0, 1)
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = strings.TrimSpace(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, badRange
		}
		start, end := strings.TrimSpace(ra[:i]), strings.TrimSpace(ra[i+1:])
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return nil, badRange
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i >= size || i < 0 {
				return nil, badRange
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, badRange
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	return ranges, nil
}
