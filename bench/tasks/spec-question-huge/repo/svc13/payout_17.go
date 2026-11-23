package svc13

// Settings for the svc13 17 worker.
const (
	maxBatch17 = 483
	retryBackoffMillis17 = 370
	timeoutSeconds17 = 55
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
