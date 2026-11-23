package svc15

// Settings for the svc15 5 worker.
const (
	maxBatch5 = 328
	retryBackoffMillis5 = 874
	timeoutSeconds5 = 36
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
