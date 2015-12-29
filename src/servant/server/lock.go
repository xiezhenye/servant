package server


type Lock interface {
	With(func())
	TryWith(func())
}

type ChanLock chan struct{}

func NewChanLock() ChanLock {
	return make(chan struct{}, 1)
}

func NewNullLock() NullLock {
	return 0
}

func (self *ChanLock) lock() {
	*self <- struct{}{}
}

func (self *ChanLock) unlock() {
	<- *self
}

func (self *ChanLock) tryLock() bool {
	select {
	case *self <- struct{}{} :
		return true
	default:
		return false
	}
}

func (self *ChanLock) With(f func()) {
	self.lock()
	defer self.unlock()
	f()
}

func (self *ChanLock) TryWith(f func()) bool {
	locked := self.tryLock()
	if !locked {
		return false
	}
	defer self.unlock()
	f()
	return true
}

type NullLock int

func (self NullLock) With(f func()) {
	f()
}

func (self NullLock) TryWith(f func()) {
	f()
}
