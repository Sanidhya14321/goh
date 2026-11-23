package svc15

// Settings for the svc15 17 worker.
const (
	maxBatch17 = 107
	retryBackoffMillis17 = 713
	timeoutSeconds17 = 37
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
