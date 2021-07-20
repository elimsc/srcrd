使用`sync.Pool`复用对象

对于`sync.Pool`管理的大小不确定的对象，当对象大小超过一定大小时，不回收
```
// Proper usage of a sync.Pool requires each entry to have approximately
// the same memory cost. To obtain this property when the stored type
// contains a variably-sized buffer, we add a hard limit on the maximum buffer
// to place back in the pool.
//
// See https://golang.org/issue/23199
if cap(p.buf) > 64<<10 {
    return
}
```