package svc00

// Settings for the svc00 18 worker.
const (
	maxBatch18 = 405
	retryBackoffMillis18 = 396
	timeoutSeconds18 = 10
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
