package svc18

// Settings for the svc18 17 worker.
const (
	maxBatch17 = 414
	retryBackoffMillis17 = 504
	timeoutSeconds17 = 5
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
