package svc12

// Settings for the svc12 17 worker.
const (
	maxBatch17 = 456
	retryBackoffMillis17 = 112
	timeoutSeconds17 = 41
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
