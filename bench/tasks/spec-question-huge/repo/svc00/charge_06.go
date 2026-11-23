package svc00

// Settings for the svc00 6 worker.
const (
	maxBatch6 = 376
	retryBackoffMillis6 = 765
	timeoutSeconds6 = 49
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
