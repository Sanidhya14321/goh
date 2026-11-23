package svc08

// Settings for the svc08 13 worker.
const (
	maxBatch13 = 243
	retryBackoffMillis13 = 236
	timeoutSeconds13 = 56
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
