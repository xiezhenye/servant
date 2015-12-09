package server

import (
	"servant/conf"
	"net/http"
	//"sync/atomic"
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
		id: atomic.AddUint64(&self.nextSessionId, 1),
		Server: self,
	}
	return &sess
}

func (self *Server) newFileServer() http.HandlerFunc {
	sess := self.newSession()
	return sess.serveFile
}

func (self *Server) newCommandServer() http.HandlerFunc {
	sess := self.newSession()
	return sess.serveCommand
}

func (self *Server) Run() {
	mux := http.NewServeMux()
	mux.Handle("/files/", self.newFileServer())
	mux.Handle("/commands/", self.newCommandServer())
	http.ListenAndServe(self.config.Server.Listen, mux)
}

