package svc12

// Settings for the svc12 5 worker.
const (
	maxBatch5 = 190
	retryBackoffMillis5 = 819
	timeoutSeconds5 = 34
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
