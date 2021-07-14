package xfmt_test

import (
	"fmt"
	"os"
	"srcrd/xfmt"
	"testing"
)

func BenchmarkStdPrint(b *testing.B) {
	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()
	os.Stdout = os.NewFile(0, os.DevNull)
	for i := 0; i < b.N; i++ {
		fmt.Print("hello\n")
	}
}

func BenchmarkPrint(b *testing.B) {
	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()
	os.Stdout = os.NewFile(0, os.DevNull)
	for i := 0; i < b.N; i++ {
		xfmt.Print([]byte("hello\n"))
	}
}

// 572 vs 542
