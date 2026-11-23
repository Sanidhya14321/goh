package svc05

// Settings for the svc05 17 worker.
const (
	maxBatch17 = 336
	retryBackoffMillis17 = 368
	timeoutSeconds17 = 15
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
