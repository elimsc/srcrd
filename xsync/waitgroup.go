package xsync

import (
	"sync/atomic"
	"unsafe"
)

type WaitGroup struct {
	noCopy noCopy

	// 64-bit value: high 32 bits are counter, low 32 bits are waiter count.
	// 64-bit atomic operations require 64-bit alignment, but 32-bit
	// compilers do not ensure it. So we allocate 12 bytes and then use
	// the aligned 8 bytes in them as state, and the other 4 as storage
	// for the sema.
	state1 [3]uint32
}

// state returns pointers to the state and sema fields stored within wg.state1.
// statep: counter, waiter, 高32bit为counter, 低32bit为waiter
func (wg *WaitGroup) state() (statep *uint64, semap *uint32) {
	// ref: https://www.luozhiyun.com/archives/429
	if uintptr(unsafe.Pointer(&wg.state1))%8 == 0 { // 判断地址是否是64位对齐, 64位的机器上起始地址被8整除(8byte)，必然是mod8的
		return (*uint64)(unsafe.Pointer(&wg.state1)), &wg.state1[2] // 内存结构为 counter, waiter, sema
	} else { // 32位机器上起始地址被4整除，不能保证一定被8整除, 那么就高4位为信号量，保证state的起始地址一定是被8整除的
		return (*uint64)(unsafe.Pointer(&wg.state1[1])), &wg.state1[0] // 内存结构为 sema, counter, waiter
	}
}

// Add adds delta, which may be negative, to the WaitGroup counter.
// If the counter becomes zero, all goroutines blocked on Wait are released.
// If the counter goes negative, Add panics.
//
// Note that calls with a positive delta that occur when the counter is zero
// must happen before a Wait. Calls with a negative delta, or calls with a
// positive delta that start when the counter is greater than zero, may happen
// at any time.
// Typically this means the calls to Add should execute before the statement
// creating the goroutine or other event to be waited for.
// If a WaitGroup is reused to wait for several independent sets of events,
// new Add calls must happen after all previous Wait calls have returned.
// See the WaitGroup example.
func (wg *WaitGroup) Add(delta int) {
	statep, semap := wg.state()

	// 如果delta>0, 增加counter值
	// 如果delta<0, 减小counter值，如果counter为0，则唤醒所有waiter

	// counter += delta
	state := atomic.AddUint64(statep, uint64(delta)<<32)
	v := int32(state >> 32) // counter
	w := uint32(state)      // waiter

	if v < 0 {
		panic("sync: negative WaitGroup counter")
	}
	if w != 0 && delta > 0 && v == int32(delta) {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	}
	// wg.Add(1)在这里就会返回了
	if v > 0 || w == 0 {
		return
	}
	// This goroutine has set counter to 0 when waiters > 0.
	// Now there can't be concurrent mutations of state:
	// - Adds must not happen concurrently with Wait,
	// - Wait does not increment waiters if it sees counter == 0.
	// Still do a cheap sanity check to detect WaitGroup misuse.
	if *statep != state {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	}
	// 所有的wg.Done()都执行后会走向这里
	// Reset waiters count to 0.
	*statep = 0
	for ; w != 0; w-- {
		// 唤醒等待的waiter
		runtime_Semrelease(semap, false, 0)
	}
}

// Wait blocks until the WaitGroup counter is zero.
func (wg *WaitGroup) Wait() {
	statep, semap := wg.state()
	for {
		state := atomic.LoadUint64(statep)
		v := int32(state >> 32) // counter
		// w := uint32(state)      // waiter
		if v == 0 {
			// Counter is 0, no need to wait.
			return
		}
		// Increment waiters count.
		if atomic.CompareAndSwapUint64(statep, state, state+1) {
			runtime_Semacquire(semap)
			if *statep != 0 {
				panic("sync: WaitGroup is reused before previous Wait has returned")
			}
			return
		}
	}
}
