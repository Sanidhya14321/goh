package svc02

// Settings for the svc02 17 worker.
const (
	maxBatch17 = 280
	retryBackoffMillis17 = 721
	timeoutSeconds17 = 32
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
