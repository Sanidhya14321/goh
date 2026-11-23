package svc17

// Settings for the svc17 17 worker.
const (
	maxBatch17 = 64
	retryBackoffMillis17 = 342
	timeoutSeconds17 = 29
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
