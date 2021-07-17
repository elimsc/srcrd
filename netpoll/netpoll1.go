package netpoll

import "atomic"

// 1. 网络轮询器的初始化；

//go:linkname poll_runtime_pollServerInit internal/poll.runtime_pollServerInit
func poll_runtime_pollServerInit() {
	netpollGenericInit()
}

func netpollGenericInit() {
	if atomic.Load(&netpollInited) == 0 { // 初始化只能调用一次
		lockInit(&netpollInitLock, lockRankNetpollInit)
		lock(&netpollInitLock)
		if netpollInited == 0 {
			netpollinit() // 会调用平台上特定实现的 runtime.netpollinit
			atomic.Store(&netpollInited, 1)
		}
		unlock(&netpollInitLock)
	}
}
