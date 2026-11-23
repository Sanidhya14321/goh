package svc11

// Settings for the svc11 10 worker.
const (
	maxBatch10 = 134
	retryBackoffMillis10 = 868
	timeoutSeconds10 = 34
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
