使用`sync.Pool`复用`gin.Context`

`var _ Interface = &Struct{}` 检测 `&Struct` 是否实现了 `Interface` 接口

路由算法 `Radix tree`, node为节点, `node.addRoute(path, handlers)` 添加节点并绑定handlers到节点上, `node.getValue(path)`查找节点并获取节点上的handlers

`context.Next()`时将context的index加1，并执行`handlers[index]`, 一个路由上的handlers的最大数量为2**7-1=127(int8的最大值)

`context.Set()`, `context.Get()`是并发安全的操作(加了读写锁) 