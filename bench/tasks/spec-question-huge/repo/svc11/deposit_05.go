package svc11

// Settings for the svc11 5 worker.
const (
	maxBatch5 = 385
	retryBackoffMillis5 = 634
	timeoutSeconds5 = 46
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
