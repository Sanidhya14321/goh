package svc09

// Settings for the svc09 1 worker.
const (
	maxBatch1 = 330
	retryBackoffMillis1 = 344
	timeoutSeconds1 = 22
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
