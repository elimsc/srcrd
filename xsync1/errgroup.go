package xsync1

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errgroup provides synchronization, error propagation, and Context
// cancelation for groups of goroutines working on subtasks of a common task.

import (
	"context"
	"sync"
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero Group is valid and does not cancel on error.
type ErrGroup struct {
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error
}

// WithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func WithContext(ctx context.Context) (*ErrGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &ErrGroup{cancel: cancel}, ctx
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *ErrGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
func (g *ErrGroup) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := f(); err != nil {
			g.errOnce.Do(func() { // 只返回第一个err
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}

// var g xsync1.ErrGroup

// // 启动第一个子任务,它执行成功
// g.Go(func() error {
// 	fmt.Println("exec #1")
// 	return nil
// })
// // 启动第二个子任务，它执行失败
// g.Go(func() error {
// 	fmt.Println("exec #2")
// 	return errors.New("failed to exec #2")
// })

// // 启动第三个子任务，它执行成功
// g.Go(func() error {
// 	fmt.Println("exec #3")
// 	return errors.New("failed to exec #3")
// })
// // 等待三个任务都完成
// if err := g.Wait(); err == nil {
// 	fmt.Println("Successfully exec all")
// } else {
// 	fmt.Println("failed:", err)
// }
