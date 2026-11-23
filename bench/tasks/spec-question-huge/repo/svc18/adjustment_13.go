package svc18

// Settings for the svc18 13 worker.
const (
	maxBatch13 = 392
	retryBackoffMillis13 = 446
	timeoutSeconds13 = 56
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
