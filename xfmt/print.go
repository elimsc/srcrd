package xfmt

import (
	"io"
	"os"
	"sync"
)

// pp is used to store a printer's state and is reused with sync.Pool to avoid allocations.
type pp struct {
	buf []byte
}

func (p *pp) doPrint(a []byte) {
	// write to buf
	p.buf = append(p.buf, a...)
}

// free saves used pp structs in ppFree; avoids an allocation per invocation.
func (p *pp) free() {
	// Proper usage of a sync.Pool requires each entry to have approximately
	// the same memory cost. To obtain this property when the stored type
	// contains a variably-sized buffer, we add a hard limit on the maximum buffer
	// to place back in the pool.
	//
	// See https://golang.org/issue/23199
	if cap(p.buf) > 64<<10 {
		return
	}

	p.buf = p.buf[:0]
	ppFree.Put(p)
}

var ppFree = sync.Pool{
	New: func() interface{} { return new(pp) },
}

func newPrinter() *pp {
	p := ppFree.Get().(*pp)
	return p
}

func Fprint(w io.Writer, a []byte) (n int, err error) {
	p := newPrinter()
	p.doPrint(a)
	n, err = w.Write(p.buf)
	p.free()
	return
}

func Print(a []byte) (n int, err error) {
	return Fprint(os.Stdout, a)
}
