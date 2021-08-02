package xsync

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// 通过 read 和 dirty 两个字段将读写分离，读的数据存在只读字段 read 上，将最新写入的数据则存在 dirty 字段上
// read和dirty中如果key相同, entry必然相同(指向同一个地址)
// 读取 read 并不需要加锁，而读或写 dirty 都需要加锁
// 有 misses 字段来统计 read 被穿透的次数（被穿透指需要读 dirty 的情况），超过一定次数(>=len(dirty))则将 dirty 数据同步到 read 上

// 读取时会先查询 read，不存在再查询 dirty，写入时则只写入 dirty
// 删除时，如果key在read中就不会执行真正的删除，而只是一个标记(key依然存在，只是value e.p = nil)
// Store时,
// 1) if read[key].p != nil && read[key].p != expunged, then read[key] = value
// 2.1) if read[key].p == expunged(key在read中，但状态为expunged), then read[key] = value, dirty[key] = value
// 2.2) if read[key].p == nil(key在read中，但状态为nil), then read[key] = value
// 3) read[key] == nil && dirty[key] != nil(key在read中没有，在dirty中), then dirty[key] = value
// 4) read[key] == nil && dirty[key] == nil(key在read和dirty中都没有), then dirty[key]=newEntry(value)(dirty不存在时创建)

// The Map type is optimized for two common use cases: (1) when the entry for a given
// key is only ever written once but read many times, as in caches that only grow,
// or (2) when multiple goroutines read, write, and overwrite entries for disjoint
// sets of keys. In these two cases, use of a Map may significantly reduce lock
// contention compared to a Go map paired with a separate Mutex or RWMutex.
// 1. key只被写一次，但需要读多次, 类似一个只变大但不更新的cache
// 2. 每个goroutine操作(读写)不同的key
type Map struct {
	mu sync.Mutex

	// read contains the portion of the map's contents that are safe for
	// concurrent access (with or without mu held).
	//
	// The read field itself is always safe to load, but must only be stored with
	// mu held.
	//
	// Entries stored in read may be updated concurrently without mu, but updating
	// a previously-expunged entry requires that the entry be copied to the dirty
	// map and unexpunged with mu held.
	read atomic.Value // readOnly

	// dirty contains the portion of the map's contents that require mu to be
	// held. To ensure that the dirty map can be promoted to the read map quickly,
	// it also includes all of the non-expunged entries in the read map.
	//
	// Expunged entries are not stored in the dirty map. An expunged entry in the
	// clean map must be unexpunged and added to the dirty map before a new value
	// can be stored to it.
	//
	// If the dirty map is nil, the next write to the map will initialize it by
	// making a shallow copy of the clean map, omitting stale entries.
	dirty map[interface{}]*entry

	// misses counts the number of loads since the read map was last updated that
	// needed to lock mu to determine whether the key was present.
	//
	// Once enough misses have occurred to cover the cost of copying the dirty
	// map, the dirty map will be promoted to the read map (in the unamended
	// state) and the next store to the map will make a new dirty copy.
	misses int
}

// readOnly is an immutable struct stored atomically in the Map.read field.
type readOnly struct {
	m map[interface{}]*entry
	// 表示 dirty 里存在 read 里没有的 key，通过该字段决定是否加锁读 dirty
	amended bool // true if the dirty map contains some key not in m.
}

// expunged is an arbitrary pointer that marks entries which have been deleted
// from the dirty map.
var expunged = unsafe.Pointer(new(interface{}))

// An entry is a slot in the map corresponding to a particular key.
type entry struct {
	// p points to the interface{} value stored for the entry.
	//
	// If p == nil, the entry has been deleted, and either m.dirty == nil or
	// m.dirty[key] is e.
	// p == nil, entry已被删除. dirty为nil, 或者dirty不为nil, 并且dirty[key] == read[key](dirty[key].p == nil)
	//
	// If p == expunged, the entry has been deleted, m.dirty != nil, and the entry
	// is missing from m.dirty.
	// p == expunged, entry已被删除，且dirty不为nil，dirty中无该key, read中有该key
	//
	// Otherwise, the entry is valid and recorded in m.read.m[key] and, if m.dirty
	// != nil, in m.dirty[key].
	// 否则, 存在于read中，如果dirty不为nil，也存在于dirty中
	//
	// An entry can be deleted by atomic replacement with nil: when m.dirty is
	// next created, it will atomically replace nil with expunged and leave
	// m.dirty[key] unset.
	//
	// An entry's associated value can be updated by atomic replacement, provided
	// p != expunged. If p == expunged, an entry's associated value can be updated
	// only after first setting m.dirty[key] = e so that lookups using the dirty
	// map find the entry.
	p unsafe.Pointer // *interface{}
}

func newEntry(i interface{}) *entry {
	return &entry{p: unsafe.Pointer(&i)}
}

func (m *Map) missLocked() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	// m.misses >= len(m.dirty)
	// 用dirty取代read, 清空dirty和misses
	m.read.Store(readOnly{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

// func (m *Map) Read() {
// 	read, _ := m.read.Load().(readOnly)
// 	fmt.Println(read.m)
// 	fmt.Println(m.dirty)
// 	fmt.Println()
// }
