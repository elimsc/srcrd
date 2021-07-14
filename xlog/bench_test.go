package xlog_test

import (
	"log"
	"srcrd/xlog"
	"testing"
)

func BenchmarkStdPrint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		log.Print("hello")
	}
}

func BenchmarkPrint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xlog.Print("hello")
	}
}
