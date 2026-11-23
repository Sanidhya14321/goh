package svc14

// Settings for the svc14 5 worker.
const (
	maxBatch5 = 492
	retryBackoffMillis5 = 378
	timeoutSeconds5 = 37
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
