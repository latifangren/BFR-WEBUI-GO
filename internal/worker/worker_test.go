package worker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerSubmit(t *testing.T) {
	var count int32
	var wg sync.WaitGroup

	numTasks := 10
	wg.Add(numTasks)

	for range numTasks {
		Submit(func() {
			atomic.AddInt32(&count, 1)
			wg.Done()
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker pool tasks")
	}

	if atomic.LoadInt32(&count) != int32(numTasks) {
		t.Errorf("expected count %d, got %d", numTasks, count)
	}

	// Submit nil should not panic
	Submit(nil)
}

func TestWorkerPanicRecovery(t *testing.T) {
	var executed int32
	var wg sync.WaitGroup
	wg.Add(2)

	// Task that panics
	Submit(func() {
		defer wg.Done()
		panic("intentional task panic")
	})

	// Subsequent task should still run
	Submit(func() {
		atomic.StoreInt32(&executed, 1)
		wg.Done()
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting after worker panic")
	}

	if atomic.LoadInt32(&executed) != 1 {
		t.Error("expected subsequent task to execute after panic")
	}
}

func TestWorkerQueueOverflowFallback(t *testing.T) {
	var count int32
	var wg sync.WaitGroup

	// Submit 150 tasks to exceed the channel capacity of 100
	numTasks := 150
	wg.Add(numTasks)

	for range numTasks {
		Submit(func() {
			atomic.AddInt32(&count, 1)
			wg.Done()
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for overflow tasks")
	}

	if atomic.LoadInt32(&count) != int32(numTasks) {
		t.Errorf("expected count %d, got %d", numTasks, count)
	}
}

func TestWorkerFallbackPanic(t *testing.T) {
	blockChan := make(chan struct{})
	var unblockWg sync.WaitGroup

	// Block the pool workers
	for range 2 {
		unblockWg.Add(1)
		Submit(func() {
			unblockWg.Done()
			<-blockChan
		})
	}
	unblockWg.Wait()

	// Fill the queue channel to 100
	for range 100 {
		Submit(func() {
			<-blockChan
		})
	}

	// Next submit will hit default: fallback goroutine. Send one that panics
	var fallbackWg sync.WaitGroup
	fallbackWg.Add(1)
	Submit(func() {
		defer fallbackWg.Done()
		panic("fallback goroutine panic")
	})

	fallbackWg.Wait()

	// Unblock all workers and drained tasks
	close(blockChan)
}
