package worker

import (
	"sync"
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
						task()
					}
				}
			}()
		}
	})
}

// Submit enqueues a background task to be processed by the 2-worker pool safely.
func Submit(fn func()) {
	if fn == nil {
		return
	}
	select {
	case globalPool.queue <- fn:
	default:
		// Queue full, execute in background goroutine as fallback
		go fn()
	}
}
