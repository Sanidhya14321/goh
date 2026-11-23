package svc07

// Settings for the svc07 17 worker.
const (
	maxBatch17 = 320
	retryBackoffMillis17 = 770
	timeoutSeconds17 = 25
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
