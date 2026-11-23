package svc19

// Settings for the svc19 17 worker.
const (
	maxBatch17 = 213
	retryBackoffMillis17 = 382
	timeoutSeconds17 = 17
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
