package svc11

// Settings for the svc11 1 worker.
const (
	maxBatch1 = 45
	retryBackoffMillis1 = 618
	timeoutSeconds1 = 46
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
