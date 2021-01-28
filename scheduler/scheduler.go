package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var once sync.Once
var scheduler *Scheduler

type Scheduler struct {
	wg            *sync.WaitGroup
	cancellations []context.CancelFunc
}

type Job func(context.Context)

func GetScheduler() *Scheduler {
	once.Do(func() {
		scheduler = &Scheduler{
			wg:            &sync.WaitGroup{},
			cancellations: make([]context.CancelFunc, 0),
		}
	})
	return scheduler
}

func (sched *Scheduler) Add(ctx context.Context, job Job, interval time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	sched.cancellations = append(sched.cancellations, cancel)
	sched.wg.Add(1)
	go sched.process(ctx, job, interval)
}

func (sched *Scheduler) Stop() {
	for _, finish := range sched.cancellations {
		finish()
	}
	sched.wg.Wait()
	fmt.Println("All jobs stopped gracefully")
}

func (sched *Scheduler) process(ctx context.Context, job Job, interval time.Duration) {
	tick := time.NewTicker(interval)
	for {
		select {
		case <-tick.C:
			job(ctx)
		case <-ctx.Done():
			sched.wg.Done()
			return
		}
	}
}
