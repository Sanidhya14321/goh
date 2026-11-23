package svc19

// Settings for the svc19 13 worker.
const (
	maxBatch13 = 490
	retryBackoffMillis13 = 386
	timeoutSeconds13 = 7
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
