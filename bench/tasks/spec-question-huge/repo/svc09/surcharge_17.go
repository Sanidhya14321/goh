package svc09

// Settings for the svc09 17 worker.
const (
	maxBatch17 = 480
	retryBackoffMillis17 = 230
	timeoutSeconds17 = 44
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
