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
	"fmt"
//	"time"
	"errors"
	"strconv"
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
	for _, allowed := range(dirConf.Allow) {
		if allowed == method {
			return nil
		}
	}
	return fmt.Errorf("method %s not allowed", method)
}

func (self *Session) openFileError(err error, method, filePath  string) {
	e := ""
	if err != nil {
		e = err.Error()
	}
	self.warn("file", "- open file %s for %s failed: %s", filePath, method, e)
	self.resp.Header().Set(ServantErrHeader, err.Error())
	self.resp.WriteHeader(http.StatusInternalServerError)
}

func (self *Session) serveFile() {
	defer self.req.Body.Close()
	method := self.req.Method
	urlPath := self.req.URL.Path
	self.info("file", "+ %s %s %s", self.req.RemoteAddr, method, urlPath)
	err := self.auth()
	if err != nil {
		self.warn("file", "- auth failed: %s", err.Error())
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}

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
	switch method {
	case "HEAD":
		file, err = os.Open(filePath)
		if err != nil {
			self.openFileError(err, method, filePath)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			self.openFileError(err, method, filePath)
			return
		}
		self.resp.Header().Add("X-Servant-File-Size", strconv.FormatInt(info.Size(), 10))
		self.resp.Header().Add("X-Servant-File-Mtime", info.ModTime().String())
		self.resp.Header().Add("X-Servant-File-Mode", info.Mode().String())
		self.resp.Header().Add("Connection", "close")
	case "GET":
		file, err = os.Open(filePath)
		if err != nil {
			self.openFileError(err, method, filePath)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			self.openFileError(err, method, filePath)
			return
		}
		rangeStr := self.req.Header.Get("Range")
		ranges, err := parseRange(rangeStr, info.Size())
		if err != nil || len(ranges) > 1 {
			self.warn("file", "- bad range format or too many ranges(%s) for file %s", rangeStr, urlPath)
			self.resp.Header().Set(ServantErrHeader, err.Error())
			self.resp.WriteHeader(http.StatusBadRequest)
			return
		}
		length := info.Size()
		if ranges != nil && len(ranges) == 1 {
			length = ranges[0].length
			if _, err = file.Seek(ranges[0].start, os.SEEK_SET); err != nil {
				self.openFileError(err, method, filePath)
				return
			}
			self.resp.Header().Set("Content-Range", ranges[0].contentRange(info.Size()))
		}
		_, err = io.CopyN(self.resp, file, length)
	case "POST":
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0664)
		if err != nil {
			self.openFileError(err, method, filePath)
			return
		}
		defer file.Close()
		_, err = io.Copy(file, self.req.Body)
	case "DELETE":
		err = os.Remove(filePath)
		if err != nil {
			self.warn("file", "- delete file %s failed", filePath)
			self.resp.Header().Set(ServantErrHeader, err.Error())
			self.resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "PUT":
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0664)
		if err != nil {
			self.openFileError(err, method, filePath)
			return
		}
		defer file.Close()
		_, err = io.Copy(file, self.req.Body)
	}
	if err != nil {
		self.warn("file", "- io error: %s", err.Error())
	} else {
		self.info("file", "- %s done", method)
	}
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
	if !strings.HasPrefix(s, b) {
		return nil, errors.New("invalid range")
	}
	var ranges []httpRange
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = strings.TrimSpace(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, errors.New("invalid range")
		}
		start, end := strings.TrimSpace(ra[:i]), strings.TrimSpace(ra[i+1:])
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return nil, errors.New("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i >= size || i < 0 {
				return nil, errors.New("invalid range")
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, errors.New("invalid range")
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



