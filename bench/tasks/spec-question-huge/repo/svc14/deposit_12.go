package svc14

// Settings for the svc14 12 worker.
const (
	maxBatch12 = 250
	retryBackoffMillis12 = 631
	timeoutSeconds12 = 46
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
