package worker

import (
	"sync"

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

// Submit enqueues a background task to be processed by the 2-worker pool safely.
func Submit(fn func()) {
	if fn == nil {
		return
	}
	select {
	case globalPool.queue <- fn:
	default:
		// Queue full, execute in background goroutine as fallback with panic recovery
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Get().Errorf("Worker", "Recovered from fallback background worker panic: %v", r)
				}
			}()
			fn()
		}()
	}
}