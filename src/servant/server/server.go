package server

import (
	"servant/conf"
	"net/http"
	"sync/atomic"
	"time"
	"regexp"
	"fmt"
)

const ServantErrHeader = "X-Servant-Err"

type Server struct {
	config          *conf.Config
	resources       map[string]HandlerFactory
	nextSessionId   uint64
}

type Session struct {
	id       uint64
	config   *conf.Config
	resource, group, item, tail string
	username string
	resp     http.ResponseWriter
	req      *http.Request
}

func NewServer(config *conf.Config) *Server {
	ret := &Server {
		config:         config,
		nextSessionId:  0,
		resources:      make(map[string]HandlerFactory),
	}
	ret.resources["commands"] = NewCommandServer
	ret.resources["files"] = NewFileServer
	ret.resources["databases"] = NewDatabaseServer
	return ret
}

func (self *Server) newSession(resp http.ResponseWriter, req *http.Request) *Session {
	resource, group, item, tail := parseUriPath(req.URL.Path)
	sess := Session {
		id:       atomic.AddUint64(&(self.nextSessionId), 1),
		config:   self.config,
		req:      req,
		resp:     resp,
		resource: resource,
		group:    group,
		item:     item,
		tail:     tail,
	}
	return &sess
}


var uriRe, _ = regexp.Compile(`^/(\w+)/(\w+)/(\w+)(/.*)?$`)
func parseUriPath(path string) (resource, group, item, tail string) {
	m := uriRe.FindStringSubmatch(path)
	if len(m) != 5 {
		return "", "", "", ""
	}
	resource, group, item, tail = m[1], m[2], m[3], m[4]
	return
}

var paramRe, _ = regexp.Compile(`\${\w+}`)


func (self *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	sess := self.newSession(resp, req)
	sess.info(sess.resource, "+ %s %s %s", req.RemoteAddr, req.Method, req.URL.Path)
	username, err := sess.auth()
	if err != nil {
		sess.ErrorEnd(http.StatusForbidden, "auth failed: %s", err)
		return
	}
	sess.username = username
	if ! sess.checkPermission() {
		sess.ErrorEnd(http.StatusForbidden, "access of %s forbidden", req.URL.Path)
		return
	}
	handlerFactory, ok := self.resources[sess.resource]
	if !ok {
		sess.ErrorEnd(http.StatusNotFound, "unknown resource")
		return
	}
	handlerFactory(sess).serve()
}

type Handler interface {
	serve()
}

type HandlerFactory func(sess *Session) Handler


func (self *Session) ErrorEnd(code int, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	self.warn(self.resource, "- " + msg)
	self.resp.Header().Set(ServantErrHeader, msg)
	self.resp.WriteHeader(code)
}

func (self *Session) BadEnd(format string, v ...interface{}) {
	self.warn(self.resource, "- " + format, v...)
}

func (self *Session) GoodEnd(format string, v ...interface{}) {
	self.info(self.resource, "- " + format, v...)
}

func (self *Session) UserConfig() *conf.User {
	ret, _ := self.config.Users[self.username]
	return ret
}

func (self *Server) Run() {
	s := &http.Server{
		Addr:           self.config.Server.Listen,
		Handler:        self,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 8192,
	}
	s.ListenAndServe()
}

