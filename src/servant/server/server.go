package server

import (
	"servant/conf"
	"net/http"
	"sync/atomic"
)

const ServantErrHeader = "X-Servant-Err"

type Server struct {
	config          *conf.Config
	nextSessionId   uint64
}

type Session struct {
	id       uint64
	*Server
	resp     http.ResponseWriter
	req      *http.Request
}

func NewServer(config *conf.Config) *Server {
	return &Server {
		config:         config,
		nextSessionId:  0,
	}
}

func (self *Server) newSession(resp http.ResponseWriter, req *http.Request) *Session {
	sess := Session{
		id:      atomic.AddUint64(&(self.nextSessionId), 1),
		Server:  self,
		req:     req,
		resp:    resp,
	}
	return &sess
}

func (self *Server) newFileServer() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sess := self.newSession(resp, req)
		sess.serveFile()
	}
}

func (self *Server) newCommandServer() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sess := self.newSession(resp, req)
		sess.serveCommand()
	}
}

func (self *Server) Run() {
	mux := http.NewServeMux()
	mux.Handle("/files/", self.newFileServer())
	mux.Handle("/commands/", self.newCommandServer())
	http.ListenAndServe(self.config.Server.Listen, mux)
}

