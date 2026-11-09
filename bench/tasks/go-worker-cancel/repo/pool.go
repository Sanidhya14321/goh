package task

import "context"

// Process squares each job with workers goroutines and returns results in job order.
func Process(ctx context.Context, jobs []int, workers int, work func(int) int) ([]int, error) {
	out := make([]int, len(jobs))
	idx := make(chan int)
	done := make(chan struct{})
	for w := 0; w < workers; w++ {
		go func() {
			for i := range idx {
				out[i] = work(jobs[i])
				done <- struct{}{}
			}
		}()
	}
	go func() {
		for i := range jobs {
			idx <- i
		}
		close(idx)
	}()
	for range jobs {
		<-done
	}
	return out, nil
}
