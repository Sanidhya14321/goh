package svc14

// Settings for the svc14 17 worker.
const (
	maxBatch17 = 449
	retryBackoffMillis17 = 264
	timeoutSeconds17 = 26
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
