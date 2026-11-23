package svc14

// Settings for the svc14 10 worker.
const (
	maxBatch10 = 310
	retryBackoffMillis10 = 121
	timeoutSeconds10 = 53
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
