```go
for {
    if c, err = acceptConn(s, ln, &lastPerIPErrorTime); err != nil {
        wp.Stop()
        if err == io.EOF {
            return nil
        }
        return err
    }
    atomic.AddInt32(&s.open, 1)
    // wp.Serve(c)就是具体处理连接的逻辑
    if !wp.Serve(c) { // 用workerpool来处理连接
        atomic.AddInt32(&s.open, -1)
        c.Close()
        panic("超出最大并发限度1024*256, TODO")
    }
    c = nil
}
```