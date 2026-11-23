package svc10

// Settings for the svc10 13 worker.
const (
	maxBatch13 = 408
	retryBackoffMillis13 = 267
	timeoutSeconds13 = 24
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
