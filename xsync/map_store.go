package xsync

import (
	"sync/atomic"
	"unsafe"
)

// m.Store(key, value)

// 情况1: create, 也就是说key既不在read中也不在dirty中
// 直接写入到dirty中(dirty不存在时先创建, 创建dirty时修改read中e.p = nil => e.p = expunged)
// 操作完后，read.amended必然为true, 即dirty中有read中不存在的key

// 情况2: update
// 如果key在read中，并且value不为expunged, 直接更新read中的值(无锁, 后面都需要锁)
// 如果key在read中，当状态为expunged, 同时更新到read和dirty
// key不在read中，而在dirty中，直接更新dirty

// 命名XXXLocked表示该操作是在Lock中执行的

// Store sets the value for a key.
func (m *Map) Store(key, value interface{}) {
	read, _ := m.read.Load().(readOnly)
	// 如果read中存在该值，并且没有被删除(expunged), 存储到read中并返回
	if e, ok := read.m[key]; ok && e.tryStore(&value) {
		return
	}

	m.mu.Lock()
	read, _ = m.read.Load().(readOnly) // 由于Lock的存在，为了获取最新状态，必须要再获取一次
	if e, ok := read.m[key]; ok {      // key在read中
		if e.unexpungeLocked() {
			// The entry was previously expunged, which implies that there is a
			// non-nil dirty map and this entry is not in it.
			// value状态为expunged, 说明存在dirty, dirty中没有key, 故写入到dirty
			m.dirty[key] = e
		}
		// 更新read
		e.storeLocked(&value)
	} else if e, ok := m.dirty[key]; ok { // key在dirty中, 更新dirty
		e.storeLocked(&value)
	} else { // key既不在read中，也不在dirty中

		if !read.amended {
			// We're adding the first new key to the dirty map.
			// Make sure it is allocated and mark the read-only map as incomplete.
			// 这里会将e.p == nil 变为 e.p == expunged
			m.dirtyLocked()
			m.read.Store(readOnly{m: read.m, amended: true})
		}
		// 写入到dirty中, 此时read.amended必然为true
		m.dirty[key] = newEntry(value)
	}
	m.mu.Unlock()
}

// tryStore stores a value if the entry has not been expunged.
//
// If the entry is expunged, tryStore returns false and leaves the entry
// unchanged.
func (e *entry) tryStore(i *interface{}) bool {
	for {
		p := atomic.LoadPointer(&e.p)
		if p == expunged {
			return false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, unsafe.Pointer(i)) {
			return true
		}
	}
}

// unexpungeLocked ensures that the entry is not marked as expunged.
//
// If the entry was previously expunged, it must be added to the dirty map
// before m.mu is unlocked.
// 如果之前是expunged, 置为nil并返回true
// 否则返回false
func (e *entry) unexpungeLocked() (wasExpunged bool) {
	return atomic.CompareAndSwapPointer(&e.p, expunged, nil)
}

// storeLocked unconditionally stores a value to the entry.
//
// The entry must be known not to be expunged.
func (e *entry) storeLocked(i *interface{}) {
	atomic.StorePointer(&e.p, unsafe.Pointer(i))
}

// 如果dirty已存在，直接返回
// 如果dirty未初始化, 初始化一个和read大小一样的dirty, 并复制read中值不为nil且不为expunge的键值对
// 调用它的时候会修改read中 e.p == nil => e.p == expunged
// 也就是说只有在dirty初始化时可能发生nil => expunged的转换
func (m *Map) dirtyLocked() {
	if m.dirty != nil {
		return
	}

	read, _ := m.read.Load().(readOnly)
	m.dirty = make(map[interface{}]*entry, len(read.m))
	for k, e := range read.m {
		// 复制read中不为nil且不为expunge的键值对, 如果e.p == nil, 会修改为e.p = expunged
		if !e.tryExpungeLocked() {
			m.dirty[k] = e
		}
	}
}

// e.p == nil, 则e.p = expunged, return true
// e.p == expunged, return true
// e.p = othervalue, return false
// 返回true时, 一定有e.p == expunged, 返回false时，一定有 e.p != expunged && e.p != nil
func (e *entry) tryExpungeLocked() (isExpunged bool) {
	p := atomic.LoadPointer(&e.p)
	for p == nil {
		if atomic.CompareAndSwapPointer(&e.p, nil, expunged) {
			return true
		}
		p = atomic.LoadPointer(&e.p)
	}
	return p == expunged
}
