package worker

import (
	"errors"
	"sync"
	"time"

	"bfr-webui-go/internal/logger"
)

type Pool struct {
	queue chan func()
	once  sync.Once
}

var globalPool = &Pool{
	queue: make(chan func(), 100),
}

func init() {
	globalPool.start(2)
}

func (p *Pool) start(numWorkers int) {
	p.once.Do(func() {
		for i := 0; i < numWorkers; i++ {
			go func() {
				for task := range p.queue {
					if task != nil {
						func() {
							defer func() {
								if r := recover(); r != nil {
									logger.Get().Errorf("Worker", "Recovered from background worker panic: %v", r)
								}
							}()
							task()
						}()
					}
				}
			}()
		}
	})
}

// ErrQueueFull is returned by Submit when no queue slot frees up in time.
var ErrQueueFull = errors.New("worker queue full")

// submitTimeout bounds how long Submit waits for a free queue slot when the
// queue is full.
const submitTimeout = 5 * time.Second

// Submit enqueues a background task to be processed by the 2-worker pool safely.
// If the queue is full it waits up to submitTimeout for a slot; if none frees
// up it returns ErrQueueFull instead of spawning an unbounded fallback
// goroutine (which under sustained load could exhaust memory on the phone).
func Submit(fn func()) error {
	if fn == nil {
		return nil
	}
	select {
	case globalPool.queue <- fn:
		return nil
	default:
	}
	select {
	case globalPool.queue <- fn:
		return nil
	case <-time.After(submitTimeout):
		return ErrQueueFull
	}
}
