package svc00

// Settings for the svc00 7 worker.
const (
	maxBatch7 = 224
	retryBackoffMillis7 = 325
	timeoutSeconds7 = 33
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
