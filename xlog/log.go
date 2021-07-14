package xlog

import (
	"io"
	"os"
	"sync"
	"time"
)

const (
	Ldate     = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                     // the time in the local time zone: 01:23:23
	LstdFlags = Ldate | Ltime // initial values for the standard logger
)

type Logger struct {
	mu   sync.Mutex // ensures atomic writes; protects the following fields
	flag int        // properties
	out  io.Writer  // destination for output
	buf  []byte     // for accumulating text to write
}

func New(out io.Writer, flag int) *Logger {
	return &Logger{out: out, flag: flag}
}

func (l *Logger) Output(s string) error {
	now := time.Now() // get this early.
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now) // set buf with time prefix
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	return err
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (l *Logger) formatHeader(buf *[]byte, t time.Time) {
	if l.flag&Ldate != 0 {
		year, month, day := t.Date()
		itoa(buf, year, 4)
		*buf = append(*buf, '/')
		itoa(buf, int(month), 2)
		*buf = append(*buf, '/')
		itoa(buf, day, 2)
		*buf = append(*buf, ' ')
	}
	if l.flag&(Ltime) != 0 {
		hour, min, sec := t.Clock()
		itoa(buf, hour, 2)
		*buf = append(*buf, ':')
		itoa(buf, min, 2)
		*buf = append(*buf, ':')
		itoa(buf, sec, 2)
		*buf = append(*buf, ' ')
	}
}

var std = New(os.Stderr, LstdFlags)

func Print(v string) {
	std.Output(v)
}
