package server

import (
	"servant/conf"
	"net/http"
	"sync/atomic"
)

type Server struct {
	config          *conf.Config
	nextSessionId   uint64
}

type Session struct {
	id      uint64
	*Server
}

func NewServer(config *conf.Config) *Server {
	return &Server {
		config: config,
	}
}

func (self *Server) newSession() *Session {
	sess := Session{
		id: atomic.AddUint64(&(self.nextSessionId), 1),
		Server: self,
	}
	return &sess
}

func (self *Server) newFileServer() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sess := self.newSession()
		sess.serveFile(resp, req)
	}
}

func (self *Server) newCommandServer() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		sess := self.newSession()
		sess.serveCommand(resp, req)
	}
}

func (self *Server) Run() {
	mux := http.NewServeMux()
	mux.Handle("/files/", self.newFileServer())
	mux.Handle("/commands/", self.newCommandServer())
	http.ListenAndServe(self.config.Server.Listen, mux)
}

