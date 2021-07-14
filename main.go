package main

import (
	"srcrd/xfmt"
	"srcrd/xlog"
)

func main() {
	xfmt.Print([]byte("hello\n"))
	xlog.Print("hello")
}
