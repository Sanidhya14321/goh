package svc01

// Settings for the svc01 17 worker.
const (
	maxBatch17 = 88
	retryBackoffMillis17 = 742
	timeoutSeconds17 = 15
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
