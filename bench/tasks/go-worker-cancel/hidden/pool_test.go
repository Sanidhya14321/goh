package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProcessAll(t *testing.T) {
	out, err := Process(context.Background(), []int{1, 2, 3, 4}, 2, func(x int) int { return x * x })
	if err != nil || out[3] != 16 || out[0] != 1 {
		t.Fatal(out, err)
	}
}

func TestProcessCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	jobs := make([]int, 1000)
	start := time.Now()
	_, err := Process(ctx, jobs, 4, func(x int) int { time.Sleep(10 * time.Millisecond); return x })
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatal(err, time.Since(start))
	}
}
