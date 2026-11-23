package svc08

// Settings for the svc08 1 worker.
const (
	maxBatch1 = 349
	retryBackoffMillis1 = 748
	timeoutSeconds1 = 44
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
