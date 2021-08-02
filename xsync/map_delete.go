package xsync

import "sync/atomic"

// Delete的时候本质是LoadAndDelete, 只要read中没有key, 就会触发miss(无论dirty中有没有)
// 如果key在read中, e.p = nil, 但不删除key
// 如果key不在read中, 删除dirty中的key

func (m *Map) Delete(key interface{}) {
	m.LoadAndDelete(key)
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *Map) LoadAndDelete(key interface{}) (value interface{}, loaded bool) {
	read, _ := m.read.Load().(readOnly)
	e, ok := read.m[key]
	if !ok && read.amended { // read中没有，并且可能在dirty中
		m.mu.Lock()
		read, _ = m.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended { // read中没有，并且可能在dirty中
			e, ok = m.dirty[key]
			delete(m.dirty, key) // 删除key
			// Regardless of whether the entry was present, record a miss: this key
			// will take the slow path until the dirty map is promoted to the read
			// map.
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if ok {
		// e可能来自read或dirty, 如果存在值, 置 e.p = nil，否则不改变e.p
		return e.delete()
	}
	return nil, false
}

// 如果e.p为nil或expunged, 直接返回nil,false, 表示该e不存在
// 否则e.p = nil, 然后返回原来的值和true
func (e *entry) delete() (value interface{}, ok bool) {
	for {
		p := atomic.LoadPointer(&e.p)
		if p == nil || p == expunged {
			return nil, false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, nil) {
			return *(*interface{})(p), true
		}
	}
}
