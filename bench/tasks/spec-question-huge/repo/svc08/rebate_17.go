package svc08

// Settings for the svc08 17 worker.
const (
	maxBatch17 = 435
	retryBackoffMillis17 = 661
	timeoutSeconds17 = 33
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
