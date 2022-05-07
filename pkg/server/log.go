package server

import (
	"fmt"
	"log"
	"os"
)

var logger = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lmicroseconds)


func (self *Session) log(topic string, level string, format string, v ...interface{}) {
	prefix := fmt.Sprintf("%s (%d) [%s] ", level, self.id, topic)
	if len(v) == 0 {
		logger.Println(prefix + format)
	} else {
		logger.Printf(prefix+format, v...)
	}
}

func (self *Session) info(format string, v ...interface{}) {
	self.log(self.resource, "INFO", format, v...)
}

func (self *Session) warn(format string, v ...interface{}) {
	self.log(self.resource, "WARN", format, v...)
}

func (self *Session) crit(format string, v ...interface{}) {
	self.log(self.resource, "CRIT", format, v...)
}
