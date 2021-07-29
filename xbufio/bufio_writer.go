package xbufio

import "io"

// Writer implements buffering for an io.Writer object.
// If an error occurs writing to a Writer, no more data will be
// accepted and all subsequent writes, and Flush, will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
type Writer struct {
	err error
	buf []byte
	n   int // 已用的buffer长度
	wr  io.Writer
}

func NewWriterSize(w io.Writer, size int) *Writer {
	// Is it already a Writer?
	b, ok := w.(*Writer)
	if ok && len(b.buf) >= size {
		return b
	}
	if size <= 0 {
		size = defaultBufSize
	}
	return &Writer{
		buf: make([]byte, size),
		wr:  w,
	}
}

// NewWriter returns a new Writer whose buffer has the default size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBufSize)
}

func (b *Writer) Write(p []byte) (nn int, err error) {
	// 1. len(p) > len(ava_buf)
	// 1.1 buf为空, 直接写入到io.Writer, buf依然为空
	// 1.2 buf有内容, 且len(p) - len(ava_buf) <=len(buf), 首先buf写满，flush，然后将剩余的p写到buf
	// 1.3 buf有内容，且len(p) - len(ava_buf) > len(buf), 首先将buf写满, flush, 然后将剩余p直接写入到io.Writer, buf为空
	// 2. len(p) <= len(ava_buf), 直接写入到buf

	// 如果当前可用buf装不下p
	for len(p) > b.Available() && b.err == nil {
		var n int
		if b.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			// buffer为空，len(p)>len(buf), 直接写入到io.Writer
			n, b.err = b.wr.Write(p)
		} else {
			// buffer不为空, 将p中的内容尽可能写入到buf(p中可能会有残余)
			n = copy(b.buf[b.n:], p)
			b.n += n
			// 然后调用flush
			b.Flush()
		}
		nn += n
		p = p[n:]
	}
	if b.err != nil {
		return nn, b.err
	}
	// 当前可用buf能装下p, 直接将p写入到buf
	n := copy(b.buf[b.n:], p)
	b.n += n
	nn += n
	return nn, nil
}

// Flush writes any buffered data to the underlying io.Writer.
func (b *Writer) Flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	// 将buf中的内容写入到实际的io.Writer
	n, err := b.wr.Write(b.buf[0:b.n])
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n {
			copy(b.buf[0:b.n-n], b.buf[n:b.n])
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

// Available returns how many bytes are unused in the buffer.
func (b *Writer) Available() int { return len(b.buf) - b.n }

// Buffered returns the number of bytes that have been written into the current buffer.
func (b *Writer) Buffered() int { return b.n }
