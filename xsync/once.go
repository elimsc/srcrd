package xsync

import (
	"sync"
	"sync/atomic"
)

type Once struct {
	done uint32
	m    sync.Mutex
}

func (o *Once) Do(f func()) {
	// o.done为1说明f已经执行过
	if atomic.LoadUint32(&o.done) == 0 {
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		// 即使f()发生panic，o.done也会变成1
		defer atomic.StoreUint32(&o.done, 1) // 在o.m.Unlock前执行，为什么不现在就执行?
		f()
	}
}
