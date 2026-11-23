package svc00

// Settings for the svc00 13 worker.
const (
	maxBatch13 = 423
	retryBackoffMillis13 = 144
	timeoutSeconds13 = 51
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
