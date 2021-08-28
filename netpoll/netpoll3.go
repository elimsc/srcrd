package netpoll

import (
	"atomic"
	"unsafe"
)

// 3. 如何从网络轮询器获取触发的事件；
// Goroutine 让出线程并等待读写事件 poll_runtime_pollWait
// 多路复用等待读写事件的发生并返回 netpoll_epoll.go/netpoll

// 等待特定的文件描述符可读/可写
// poll_runtime_pollWait, which is internal/poll.runtime_pollWait,
// waits for a descriptor to be ready for reading or writing,
// according to mode, which is 'r' or 'w'.
// This returns an error code; the codes are defined above.
//go:linkname poll_runtime_pollWait internal/poll.runtime_pollWait
func poll_runtime_pollWait(pd *pollDesc, mode int) int {
	errcode := netpollcheckerr(pd, int32(mode))
	if errcode != pollNoError {
		return errcode
	}
	// As for now only Solaris, illumos, and AIX use level-triggered IO.
	if GOOS == "solaris" || GOOS == "illumos" || GOOS == "aix" {
		netpollarm(pd, mode)
	}
	for !netpollblock(pd, int32(mode), false) { // 等待
		errcode = netpollcheckerr(pd, int32(mode))
		if errcode != pollNoError {
			return errcode
		}
		// Can happen if timeout has fired and unblocked us,
		// but before we had a chance to run, timeout has been reset.
		// Pretend it has not happened and retry.
	}
	return pollNoError
}

// returns true if IO is ready, or false if timedout or closed
// waitio - wait only for completed IO, ignore errors
// Goroutine 等待 I/O 事件的关键函数，它会使用 runtime.gopark 让出当前线程，将 Goroutine 转换到休眠状态并等待运行时的唤醒
func netpollblock(pd *pollDesc, mode int32, waitio bool) bool {
	gpp := &pd.rg
	if mode == 'w' {
		gpp = &pd.wg
	}

	// set the gpp semaphore to pdWait
	for {
		old := *gpp
		if old == pdReady {
			*gpp = 0
			return true
		}
		if old != 0 {
			throw("runtime: double wait")
		}
		if atomic.Casuintptr(gpp, 0, pdWait) {
			break
		}
	}

	// need to recheck error states after setting gpp to pdWait
	// this is necessary because runtime_pollUnblock/runtime_pollSetDeadline/deadlineimpl
	// do the opposite: store to closing/rd/wd, membarrier, load of rg/wg
	if waitio || netpollcheckerr(pd, mode) == 0 {
		gopark(netpollblockcommit, unsafe.Pointer(gpp), waitReasonIOWait, traceEvGoBlockNet, 5)
	}
	// be careful to not lose concurrent pdReady notification
	old := atomic.Xchguintptr(gpp, 0)
	if old > pdWait {
		throw("runtime: corrupted polldesc")
	}
	return old == pdReady
}

// 在netpoll_epoll.go中的nepoll中调用
// 获取pd上等待的wg(写事件),rg(读事件), 并将G加入到toRun列表
// netpollready is called by the platform-specific netpoll function.
// It declares that the fd associated with pd is ready for I/O.
// The toRun argument is used to build a list of goroutines to return
// from netpoll. The mode argument is 'r', 'w', or 'r'+'w' to indicate
// whether the fd is ready for reading or writing or both.
//
// This may run while the world is stopped, so write barriers are not allowed.
//go:nowritebarrier
func netpollready(toRun *gList, pd *pollDesc, mode int32) {
	var rg, wg *g
	if mode == 'r' || mode == 'r'+'w' {
		rg = netpollunblock(pd, 'r', true)
	}
	if mode == 'w' || mode == 'r'+'w' {
		wg = netpollunblock(pd, 'w', true)
	}
	if rg != nil {
		toRun.push(rg)
	}
	if wg != nil {
		toRun.push(wg)
	}
}

// 获取pd上等待的g，同时会修改pd的内部状态rg,wg
func netpollunblock(pd *pollDesc, mode int32, ioready bool) *g {
	gpp := &pd.rg
	if mode == 'w' {
		gpp = &pd.wg
	}

	for {
		old := *gpp
		if old == pdReady {
			return nil
		}
		if old == 0 && !ioready {
			// Only set pdReady for ioready. runtime_pollWait
			// will check for timeout/cancel before waiting.
			return nil
		}
		var new uintptr
		if ioready {
			new = pdReady
		}
		if atomic.Casuintptr(gpp, old, new) {
			if old == pdWait {
				old = 0
			}
			return (*g)(unsafe.Pointer(old))
		}
	}
}

func netpollcheckerr(pd *pollDesc, mode int32) int {
	if pd.closing {
		return pollErrClosing
	}
	if (mode == 'r' && pd.rd < 0) || (mode == 'w' && pd.wd < 0) {
		return pollErrTimeout
	}
	// Report an event scanning error only on a read event.
	// An error on a write event will be captured in a subsequent
	// write call that is able to report a more specific error.
	if mode == 'r' && pd.everr {
		return pollErrNotPollable
	}
	return pollNoError
}
