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
		for _, pattern := range(dirConf.Patterns) {
			ok, err := regexp.MatchString(pattern, relPath)
			if err != nil && ok {
				break
			}
		}
		if ! ok {
			return fmt.Errorf("%s not match allowed pattern", relPath, method)
		}
	}
	return nil
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

func (self *Session) serveGetFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		self.openFileError(err, "GET", filePath)
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		self.openFileError(err, "GET", filePath)
		return err
	}
	rangeStr := self.req.Header.Get("Range")
	ranges, err := parseRange(rangeStr, info.Size())
	if err != nil || len(ranges) > 1 {
		self.warn("file", "- bad range format or too many ranges(%s)", rangeStr)
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusBadRequest)
		return err
	}
	length := info.Size()
	if ranges != nil && len(ranges) == 1 {
		length = ranges[0].length
		if _, err = file.Seek(ranges[0].start, os.SEEK_SET); err != nil {
			self.openFileError(err, "GET", filePath)
			return err
		}
		self.resp.Header().Set("Content-Range", ranges[0].contentRange(info.Size()))
	}
	_, err = io.CopyN(self.resp, file, length)
	return err
}

func (self *Session) serveHeadFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		self.openFileError(err, "HEAD", filePath)
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		self.openFileError(err, "HEAD", filePath)
		return err
	}
	self.resp.Header().Add("X-Servant-File-Size", strconv.FormatInt(info.Size(), 10))
	self.resp.Header().Add("X-Servant-File-Mtime", info.ModTime().String())
	self.resp.Header().Add("X-Servant-File-Mode", info.Mode().String())
	self.resp.Header().Add("Connection", "close")
	return nil
}

func (self *Session) servePostFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0664)
	if err != nil {
		self.openFileError(err, "POST", filePath)
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, self.req.Body)
	return err
}

func (self *Session) servePutFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		self.openFileError(err, "PUT", filePath)
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, self.req.Body)
	return err
}

func (self *Session) serveDeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		self.warn("file", "- delete file %s failed", filePath)
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusInternalServerError)
		return err
	}
	return nil
}

func (self *Session) serveFile() {
	defer self.req.Body.Close()
	method := self.req.Method
	urlPath := self.req.URL.Path
	self.info("file", "+ %s %s %s", self.req.RemoteAddr, method, urlPath)
	var err error
	if err = self.auth(); err != nil {
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
	if err = checkDirAllow(dirConf, relPath, method); err != nil {
		self.warn("file", "- %s", err.Error())
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}
	filePath := path.Clean(dirConf.Root + relPath)
	if ! strings.HasPrefix(filePath, path.Clean(dirConf.Root) + "/") {
		self.warn("file", "- attempt to %s out of root: %s", method, relPath)
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}
	switch method {
	case "HEAD":
		err = self.serveHeadFile(filePath)
	case "GET":
		err = self.serveGetFile(filePath)
	case "POST":
		err = self.servePostFile(filePath)
	case "DELETE":
		err = self.serveDeleteFile(filePath)
	case "PUT":
		err = self.servePutFile(filePath)
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
