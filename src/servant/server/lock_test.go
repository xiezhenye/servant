package server

import (
	"testing"
	"time"
	"sync"
)

func TestLock(t *testing.T) {
	lock := NewChanLock()
	lock.With(func(){
		//
	})
	if ! lock.TryWith(func(){}) {
		t.Error("Try should ok")
	}
	var wg  sync.WaitGroup
	out := make([]int, 0, 2)
	wg.Add(2)
	go func() {
		time.Sleep(200 * time.Millisecond)
		lock.With(func(){
			out = append(out, 2)
		})
		wg.Done()
	}()
	go func() {
		time.Sleep(100 * time.Millisecond)
		lock.With(func(){
			time.Sleep(200 * time.Millisecond)
			out = append(out, 1)
		})
		wg.Done()
	}()
	wg.Wait()
	if out[0] != 1 || out[1] != 2 {
		t.Error("sync failed")
	}

	go lock.With(func() {
		time.Sleep(200 * time.Millisecond)
	})
	time.Sleep(100 * time.Millisecond)
	if lock.TryWith(func() { }) != false {
		t.Error("try should fail")
	}
	time.Sleep(200 * time.Millisecond)


	go lock.TryWith(func() {
		time.Sleep(200 * time.Millisecond)
	})
	time.Sleep(100 * time.Millisecond)
	if lock.TryWith(func() { }) != false {
		t.Error("try should fail")
	}
	time.Sleep(200 * time.Millisecond)
}
