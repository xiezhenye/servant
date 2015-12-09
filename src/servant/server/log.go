package server
import (
	"os"
	"log"
	"fmt"
)
var logger = log.New(os.Stdout, "", log.LstdFlags)

func (self *Session) log(topic string, level string, format string, v ...interface{}) {
	prefix := fmt.Sprintf("%s (%d) [%s] ", level, self.id, topic)
	logger.Printf(prefix + format, v...)
}

func (self *Session) info(topic string, format string, v ...interface{}) {
	self.log(topic, "INFO", format, v...)
}

func (self *Session) warn(topic string, format string, v ...interface{}) {
	self.log(topic, "WARN", format, v...)
}

func (self *Session) crit(topic string, format string, v ...interface{}) {
	self.log(topic, "CRIT", format, v...)
}
