package svc04

// Settings for the svc04 13 worker.
const (
	maxBatch13 = 44
	retryBackoffMillis13 = 791
	timeoutSeconds13 = 60
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
