package netpoll

import "unsafe"

// func netpollinit()
//     Initialize the poller. Only called once.
//     初始化poller, 只调用一次
//
// func netpollopen(fd uintptr, pd *pollDesc) int32
//     Arm edge-triggered notifications for fd. The pd argument is to pass
//     back to netpollready when fd is ready. Return an errno value.
//     监听文件描述符上的边缘触发事件，创建事件并加入监听
//
// func netpollclose(fd uintptr) int32
//     Disable notifications for fd. Return an errno value.
//     	取消监听文件描述符fd
//
// func netpoll(delta int64) gList
//     Poll the network. If delta < 0, block indefinitely. If delta == 0,
//     poll without blocking. If delta > 0, block for up to delta nanoseconds.
//     Return a list of goroutines built by calling netpollready.
//     轮询网络并返回一组已经准备就绪的 Goroutine
//
// func netpollBreak()
//     Wake up the network poller, assumed to be blocked in netpoll.
//     唤醒poller
//
// func netpollIsPollDescriptor(fd uintptr) bool
//     Reports whether fd is a file descriptor used by the poller.
//     判断文件描述符是否被轮询器使用

const (
	pollNoError        = 0 // no error
	pollErrClosing     = 1 // descriptor is closed
	pollErrTimeout     = 2 // I/O timeout
	pollErrNotPollable = 3 // general error polling descriptor
)

// pollDesc contains 2 binary semaphores, rg and wg, to park reader and writer
// goroutines respectively. The semaphore can be in the following states:
// 2个二进制信号量rg, wg, 可能的状态如下:
// pdReady - io readiness notification is pending;
//           a goroutine consumes the notification by changing the state to nil.
//           io就绪
// pdWait - a goroutine prepares to park on the semaphore, but not yet parked;
//          the goroutine commits to park by changing the state to G pointer,
//          or, alternatively, concurrent io notification changes the state to pdReady,
//          or, alternatively, concurrent timeout/close changes the state to nil.
//          中间状态, 有G准备park, 而未park时
//          pdWait -> G pointer 发生在有G park 在信号量上时
//          pdWait -> pdReady 发生在有io就绪时。准备的G不park，直接处理io
//          pdWait -> nil 发生在timeout/close时
// G pointer - the goroutine is blocked on the semaphore;
//             io notification or timeout/close changes the state to pdReady or nil respectively
//             and unparks the goroutine.
// nil - none of the above.
const (
	pdReady uintptr = 1
	pdWait  uintptr = 2
)

const pollBlockSize = 4 * 1024

type timer struct{}
type mutex struct{}

type pollDesc struct {
	link *pollDesc // in pollcache, protected by pollcache.lock

	// The lock protects pollOpen, pollSetDeadline, pollUnblock and deadlineimpl operations.
	// This fully covers seq, rt and wt variables. fd is constant throughout the PollDesc lifetime.
	// pollReset, pollWait, pollWaitCanceled and runtime·netpollready (IO readiness notification)
	// proceed w/o taking the lock. So closing, everr, rg, rd, wg and wd are manipulated
	// in a lock-free way by all operations.
	// NOTE(dvyukov): the following code uses uintptr to store *g (rg/wg),
	// that will blow up when GC starts moving objects.
	lock    mutex // protects the following fields
	fd      uintptr
	closing bool
	everr   bool      // marks event scanning error happened
	user    uint32    // user settable cookie
	rseq    uintptr   // protects from stale read timers 表示文件描述符被重用或者计时器被重置
	rg      uintptr   // pdReady, pdWait, G waiting for read or nil
	rt      timer     // read deadline timer (set if rt.f != nil) 等待文件描述符的计时器
	rd      int64     // read deadline 等待文件描述符可读的截止日期
	wseq    uintptr   // protects from stale write timers
	wg      uintptr   // pdReady, pdWait, G waiting for write or nil
	wt      timer     // write deadline timer
	wd      int64     // write deadline
	self    *pollDesc // storage for indirect interface. See (*pollDesc).makeArg.
}

// 可用的pollDesc组成的单链表(复用pollDesc; 当无可用pollDesc时，向操作系统申请)
type pollCache struct {
	lock  mutex
	first *pollDesc
}

var (
	netpollInitLock mutex
	netpollInited   uint32 // 判断poller是否初始化, 因为netpoll的初始化只能调用一次

	pollcache      pollCache // 全局变量
	netpollWaiters uint32
)

// 返回链表头还没有被使用的 runtime.pollDesc
// 当无可用pollDesc时，向操作系统申请
func (c *pollCache) alloc() *pollDesc {
	lock(&c.lock)
	if c.first == nil {
		const pdSize = unsafe.Sizeof(pollDesc{})
		n := pollBlockSize / pdSize // 一次分配n个pollDesc的内存
		if n == 0 {
			n = 1
		}
		// Must be in non-GC memory because can be referenced
		// only from epoll/kqueue internals.
		mem := persistentalloc(n*pdSize, 0, &memstats.other_sys)
		for i := uintptr(0); i < n; i++ {
			pd := (*pollDesc)(add(mem, i*pdSize))
			pd.link = c.first
			c.first = pd
		}
	}
	pd := c.first
	c.first = pd.link
	lockInit(&pd.lock, lockRankPollDesc)
	unlock(&c.lock)
	return pd
}

// pd插入到链表头
func (c *pollCache) free(pd *pollDesc) {
	lock(&c.lock)
	pd.link = c.first
	c.first = pd
	unlock(&c.lock)
}
