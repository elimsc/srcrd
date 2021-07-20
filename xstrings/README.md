`strings.Builder`转string, 使用`unsafe.Pointer`避免copy

`builder.Grow(n)`, 现在空余空间不足n时, 变为`cap()*2+n`