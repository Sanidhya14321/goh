package task

import (
	"context"
	"sync"
)

// Process squares each job with workers goroutines and returns results in job order.
func Process(ctx context.Context, jobs []int, workers int, work func(int) int) ([]int, error) {
	out := make([]int, len(jobs))
	idx := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				out[i] = work(jobs[i])
			}
		}()
	}
feed:
	for i := range jobs {
		select {
		case idx <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(idx)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
