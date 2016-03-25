package server

import (
	"servant/conf"
	"net/http"
	"sync/atomic"
	"time"
	"regexp"
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

func (self *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	sess := self.newSession(resp, req)
	handlerFactory, ok := self.resources[sess.resource]
	if !ok {
		return
	}
	handlerFactory(sess).serve()
}

type Handler interface {
	serve()
}

type HandlerFactory func(sess *Session) Handler

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

