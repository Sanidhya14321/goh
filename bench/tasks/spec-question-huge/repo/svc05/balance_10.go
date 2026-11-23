package svc05

// Settings for the svc05 10 worker.
const (
	maxBatch10 = 145
	retryBackoffMillis10 = 218
	timeoutSeconds10 = 11
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
