package xsync

// 垃圾回收时，调用该函数. gcStart中调用clearpools时，会调用此函数
// pool需要两次GC才能完全清除掉，第一次GC将local变为victim, 第二次GC将victim完全删除
func poolCleanup() {
	// This function is called with the world stopped, at the beginning of a garbage collection.
	// It must not allocate and probably should not call any runtime functions.

	// Because the world is stopped, no pool user can be in a
	// pinned section (in effect, this has all Ps pinned).

	// Drop victim caches from all pools.
	// 清空oldPools中的victim
	for _, p := range oldPools {
		p.victim = nil
		p.victimSize = 0
	}

	// Move primary cache to victim cache.
	// 将local赋给victim，并置local为nil
	for _, p := range allPools {
		p.victim = p.local
		p.victimSize = p.localSize
		p.local = nil
		p.localSize = 0
	}

	// The pools with non-empty primary caches now have non-empty
	// victim caches and no pools have primary caches.
	// 将allPools赋给oldPools, 并置allPools为nil
	oldPools, allPools = allPools, nil
}
