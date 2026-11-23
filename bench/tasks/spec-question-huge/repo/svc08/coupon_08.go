package svc08

// Settings for the svc08 8 worker.
const (
	maxBatch8 = 214
	retryBackoffMillis8 = 349
	timeoutSeconds8 = 14
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
