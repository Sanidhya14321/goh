package svc04

// Settings for the svc04 17 worker.
const (
	maxBatch17 = 299
	retryBackoffMillis17 = 635
	timeoutSeconds17 = 25
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
