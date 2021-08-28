为什么一定要使用非阻塞 I/O?  
阻塞IO会导致线程阻塞进入内核态，那么1w个连接就要1w个线程。使用非阻塞IO，当返回EAGIN时，通过gopark方法把当前的G休眠进入休眠，从而没有进入到内核态，都是由Go的runtime来调度的

如何做到代码同步而底层异步的?  
因为Go中read, write, accept等阻塞时，是G的park，而不是物理线程的阻塞。Go通过netpoll调用操作系统的epollwait获取有事件发生的socket，然后获取socket对应的G，然后将G加入到全局可运行队列中。  
也就是说，操作返回EAGIN时就park当前G，G的唤醒(即加入到全局可运行队列)由守护线程中一直运行的netpoll来负责。

通过netpoll轮询可运行的G
```go
n := epollwait(epfd, &events[0], int32(len(events)), waitms)
for i := int32(0); i < n; i++ {
    if mode != 0 {
        pd := *(**pollDesc)(unsafe.Pointer(&ev.data))
        // 获取pd上等待的g到toRun中, toRun中的G会被加入到全局的可运行队列中
        netpollready(&toRun, pd, mode)
    }
}
return toRun
```

一个TCP Server的例子
```go
// nc 127.0.0.1 3000
l, _ := net.Listen("tcp", ":3000")
for {
	// 1. 调用runtime_pollWait来park当前G
	// 2. G被唤醒后，继续执行, 获得新连接对应的newfd
	// 3. 然后把新连接交由netpoll管理。runtime_pollServerInit(once), runtime_pollOpen(newfd)
	conn, _ := l.Accept()
	go func() {
		b := make([]byte, 100)
        // 调用runtime_pollWait来park当前G, 直到数据可读，然后当前G被唤醒
		conn.Read(b)
        // 调用runtime_pollWait来park当前G, 直到可写，但后当前G被唤醒
		conn.Write(b)

		conn.Close()
	}()
}
```

accept
```go
func (fd *netFD) accept() (netfd *netFD, err error) {
    d, rsa, errcall, err := fd.pfd.Accept() // runtime_pollWait
    netfd, err = newFD(d, fd.family, fd.sotype, fd.net) // 创建一个新的socket
    netfd.init() // runtime_pollServerInit(once), runtime_pollOpen

    lsa, _ := syscall.Getsockname(netfd.pfd.Sysfd)
    netfd.setAddr(netfd.addrFunc()(lsa), netfd.addrFunc()(rsa))
    return netfd, nil
}

// fd.pfd.Accept
func (fd *FD) Accept() (int, syscall.Sockaddr, string, error) {
	if err := fd.readLock(); err != nil {
		return -1, nil, "", err
	}
	defer fd.readUnlock()

	if err := fd.pd.prepareRead(fd.isFile); err != nil {
		return -1, nil, "", err
	}
	for {
        // 使用 linux 系统调用 accept 接收新连接，创建对应的 socket
		s, rsa, errcall, err := accept(fd.Sysfd)
        // 因为 listener fd 在创建的时候已经设置成非阻塞的了，
		// 所以 accept 方法会直接返回，不管有没有新连接到来；如果 err == nil 则表示正常建立新连接，直接返回
		if err == nil {
			return s, rsa, "", err
		}
		switch err {
		case syscall.EINTR:
			continue
		case syscall.EAGAIN:
			if fd.pd.pollable() {
                // 如果当前没有发生期待的 I/O 事件，那么 waitRead 会通过 park goroutine 让逻辑 block 在这里
                // runtime_pollWait(pd.runtimeCtx, 'r')
				if err = fd.pd.waitRead(fd.isFile); err == nil {
					continue
				}
			}
		case syscall.ECONNABORTED:
			// This means that a socket on the listen
			// queue was closed before we Accept()ed it;
			// it's a silly error, so try again.
			continue
		}
		return -1, nil, errcall, err
	}
}
```

read: 调用runtime_pollWait(pd.runtimeCtx, 'r')
```go
func (fd *FD) Read(p []byte) (int, error) {
	if err := fd.readLock(); err != nil {
		return 0, err
	}
	defer fd.readUnlock()
	if len(p) == 0 {
		// If the caller wanted a zero byte read, return immediately
		// without trying (but after acquiring the readLock).
		// Otherwise syscall.Read returns 0, nil which looks like
		// io.EOF.
		// TODO(bradfitz): make it wait for readability? (Issue 15735)
		return 0, nil
	}
	if err := fd.pd.prepareRead(fd.isFile); err != nil {
		return 0, err
	}
	if fd.IsStream && len(p) > maxRW {
		p = p[:maxRW]
	}
	for {
		n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
		if err != nil {
			n = 0
			if err == syscall.EAGAIN && fd.pd.pollable() {
                // runtime_pollWait(pd.runtimeCtx, 'r')
				if err = fd.pd.waitRead(fd.isFile); err == nil {
					continue
				}
			}
		}
		err = fd.eofError(n, err)
		return n, err
	}
}
```

write: 调用runtime_pollWait(pd.runtimeCtx, 'w')
```go
func (fd *FD) Write(p []byte) (int, error) {
	if err := fd.writeLock(); err != nil {
		return 0, err
	}
	defer fd.writeUnlock()
	if err := fd.pd.prepareWrite(fd.isFile); err != nil {
		return 0, err
	}
	var nn int
	for {
		max := len(p)
		if fd.IsStream && max-nn > maxRW {
			max = nn + maxRW
		}
		n, err := ignoringEINTRIO(syscall.Write, fd.Sysfd, p[nn:max])
		if n > 0 {
			nn += n
		}
		if nn == len(p) {
			return nn, err
		}
		if err == syscall.EAGAIN && fd.pd.pollable() {
            // runtime_pollWait(pd.runtimeCtx, 'w')
			if err = fd.pd.waitWrite(fd.isFile); err == nil {
				continue
			}
		}
		if err != nil {
			return nn, err
		}
		if n == 0 {
			return nn, io.ErrUnexpectedEOF
		}
	}
}
```