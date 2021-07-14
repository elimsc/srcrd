package main

import (
	"srcrd/xlog"
	"srcrd/xstrings"
)

func main() {
	var builder xstrings.Builder
	builder.Write([]byte("hello"))
	xlog.Print(builder.String())

	s := xstrings.Join([]string{"1", "2"}, ",")
	xlog.Print(s)
}
