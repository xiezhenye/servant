package server

import (
	"sync"
	"time"
)

type Lock interface {
	With(func())
	TryWith(func()) bool
	TimeoutWith(d time.Duration, f func()) bool
}

type ChanLock chan struct{}

func NewChanLock() ChanLock {
	return make(chan struct{}, 1)
}

var locks = make(map[string]Lock)
var lockMapMutex sync.Mutex

func GetLock(name string) Lock {
	lockMapMutex.Lock()
	defer lockMapMutex.Unlock()
	lock, ok := locks[name]
	if ok {
		return lock
	}
	lock = NewChanLock()
	locks[name] = lock
	return lock
}

func (self ChanLock) lock() {
	self <- struct{}{}
}

func (self ChanLock) unlock() {
	<-self
}

func (self ChanLock) tryLock() bool {
	select {
	case self <- struct{}{}:
		return true
	default:
		return false
	}
}

func (self ChanLock) timeoutLock(d time.Duration) bool {
	select {
	case self <- struct{}{}:
		return true
	case <-time.After(d):
		return false
	}
}

func (self ChanLock) With(f func()) {
	self.lock()
	defer self.unlock()
	f()
}

func (self ChanLock) TryWith(f func()) bool {
	locked := self.tryLock()
	if !locked {
		return false
	}
	defer self.unlock()
	f()
	return true
}

func (self ChanLock) TimeoutWith(d time.Duration, f func()) bool {
	locked := self.timeoutLock(d)
	if !locked {
		return false
	}
	defer self.unlock()
	f()
	return true
}
