package xsync

import (
	"sync"
	"sync/atomic"
)

// There is a modified copy of this file in runtime/rwmutex.go.
// If you make any changes here, see if you should make them there.

// A RWMutex is a reader/writer mutual exclusion lock.
// The lock can be held by an arbitrary number of readers or a single writer.
// The zero value for a RWMutex is an unlocked mutex.
//
// A RWMutex must not be copied after first use.
//
// If a goroutine holds a RWMutex for reading and another goroutine might
// call Lock, no goroutine should expect to be able to acquire a read lock
// until the initial read lock is released. In particular, this prohibits
// recursive read locking. This is to ensure that the lock eventually becomes
// available; a blocked Lock call excludes new readers from acquiring the
// lock.
// 加锁顺序: 读 读 写 读 读. 写锁后面的读操作必须要等待写操作完成后才能继续持有读锁, 等待中的写操作会阻塞后面的所有读操作
type RWMutex struct {
	w         sync.Mutex // held if there are pending writers
	writerSem uint32     // semaphore for writers to wait for completing readers
	readerSem uint32     // semaphore for readers to wait for completing writers

	// 该值小于0时，表示有writer在等待或执行.
	readerCount int32 // number of pending readers, 持有RLock的Reader数量,

	// 当有writer等待时，该值大于0, 表示writer还需要等待的reader数量. 该值由大于0变为0时，唤醒等待的writer
	readerWait int32 // number of departing readers,

	// readerCount == 0, 无reader与writer
	// readerCount > 0, 无writer等待, 有reader在执行
	// readerCount < 0 && readerWait > 0, 有reader在执行, 有writer在等待
	// readerCount < 0 && readerWait == 0, 有writer在执行
}

const rwmutexMaxReaders = 1 << 30

// Happens-before relationships are indicated to the race detector via:
// - Unlock  -> Lock:  readerSem
// - Unlock  -> RLock: readerSem
// - RUnlock -> Lock:  writerSem
//
// The methods below temporarily disable handling of race synchronization
// events in order to provide the more precise model above to the race
// detector.
//
// For example, atomic.AddInt32 in RLock should not appear to provide
// acquire-release semantics, which would incorrectly synchronize racing
// readers, thus potentially missing races.

// RLock locks rw for reading.
//
// It should not be used for recursive read locking; a blocked Lock
// call excludes new readers from acquiring the lock. See the
// documentation on the RWMutex type.
func (rw *RWMutex) RLock() {
	if atomic.AddInt32(&rw.readerCount, 1) < 0 {
		// A writer is pending, wait for it.
		runtime_SemacquireMutex(&rw.readerSem, false, 0)
	}
}

// RUnlock undoes a single RLock call;
// it does not affect other simultaneous readers.
// It is a run-time error if rw is not locked for reading
// on entry to RUnlock.
func (rw *RWMutex) RUnlock() {
	if r := atomic.AddInt32(&rw.readerCount, -1); r < 0 {
		// Outlined slow-path to allow the fast-path to be inlined
		// r < 0 说明有writer在等待, 需要去唤醒它
		rw.rUnlockSlow(r)
	}
}

func (rw *RWMutex) rUnlockSlow(r int32) {
	if r+1 == 0 || r+1 == -rwmutexMaxReaders {
		// race.Enable()
		panic("sync: RUnlock of unlocked RWMutex")
	}
	// A writer is pending.
	if atomic.AddInt32(&rw.readerWait, -1) == 0 {
		// The last reader unblocks the writer.
		runtime_Semrelease(&rw.writerSem, false, 1)
	}
}

// Lock locks rw for writing.
// If the lock is already locked for reading or writing,
// Lock blocks until the lock is available.
func (rw *RWMutex) Lock() {
	// First, resolve competition with other writers.
	// 如果有其它writer，就会阻塞在这里
	rw.w.Lock()
	// Announce to readers there is a pending writer.
	// 1. r = rw.readerCount, r为现有的持有RLock的Reader数量
	// 2. rw.readerCount -= rwmutexMaxReaders, 现在rw.readerCount一定是小于0的, 那么后面来的reader都只能等待
	r := atomic.AddInt32(&rw.readerCount, -rwmutexMaxReaders) + rwmutexMaxReaders
	// Wait for active readers.
	// rw.readerWait += r, 如果有Reader正在操作, 那么r > 0, rw.readerWait > 0
	if r != 0 && atomic.AddInt32(&rw.readerWait, r) != 0 {
		runtime_SemacquireMutex(&rw.writerSem, false, 0)
	}

}

// Unlock unlocks rw for writing. It is a run-time error if rw is
// not locked for writing on entry to Unlock.
//
// As with Mutexes, a locked RWMutex is not associated with a particular
// goroutine. One goroutine may RLock (Lock) a RWMutex and then
// arrange for another goroutine to RUnlock (Unlock) it.
func (rw *RWMutex) Unlock() {
	// Announce to readers there is no active writer.
	r := atomic.AddInt32(&rw.readerCount, rwmutexMaxReaders)
	if r >= rwmutexMaxReaders {
		// race.Enable()
		panic("sync: Unlock of unlocked RWMutex")
	}
	// r为当前等待中的reader数量
	// Unblock blocked readers, if any.
	for i := 0; i < int(r); i++ {
		runtime_Semrelease(&rw.readerSem, false, 0)
	}
	// Allow other writers to proceed.
	rw.w.Unlock()
}
