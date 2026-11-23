package svc16

// Settings for the svc16 17 worker.
const (
	maxBatch17 = 204
	retryBackoffMillis17 = 424
	timeoutSeconds17 = 45
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
