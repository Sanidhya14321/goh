package svc11

// Settings for the svc11 17 worker.
const (
	maxBatch17 = 53
	retryBackoffMillis17 = 347
	timeoutSeconds17 = 51
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
