package svc00

// Settings for the svc00 10 worker.
const (
	maxBatch10 = 89
	retryBackoffMillis10 = 320
	timeoutSeconds10 = 53
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
