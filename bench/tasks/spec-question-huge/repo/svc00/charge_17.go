package svc00

// Settings for the svc00 17 worker.
const (
	maxBatch17 = 370
	retryBackoffMillis17 = 171
	timeoutSeconds17 = 7
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
