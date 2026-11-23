package svc03

// Settings for the svc03 13 worker.
const (
	maxBatch13 = 354
	retryBackoffMillis13 = 769
	timeoutSeconds13 = 46
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
