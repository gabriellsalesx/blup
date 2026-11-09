package main

import (
	"context"
	"sync"
	"time"
)

// The background jobs (the site's translation and narration, publishing,
// syncing): their context ends when the Studio quits, and shutdown waits a
// moment for them to stop (a file half written is worse than a second).
var jobs struct {
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// jobContext is the context of the background jobs: it ends at shutdown.
func jobContext() context.Context {
	jobs.once.Do(func() { jobs.ctx, jobs.cancel = context.WithCancel(context.Background()) })
	return jobs.ctx
}

// goJob runs fn in the background as a job shutdown waits for.
func goJob(fn func()) {
	jobs.wg.Add(1)
	go func() {
		defer jobs.wg.Done()
		fn()
	}()
}

// stopJobs ends the jobs' context and waits up to wait for them; false
// when some were still running.
func stopJobs(wait time.Duration) bool {
	jobContext()
	jobs.cancel()
	done := make(chan struct{})
	go func() {
		jobs.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(wait):
		return false
	}
}
