package svc12

// Settings for the svc12 10 worker.
const (
	maxBatch10 = 151
	retryBackoffMillis10 = 841
	timeoutSeconds10 = 42
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
