package svc06

// Settings for the svc06 17 worker.
const (
	maxBatch17 = 405
	retryBackoffMillis17 = 385
	timeoutSeconds17 = 27
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
