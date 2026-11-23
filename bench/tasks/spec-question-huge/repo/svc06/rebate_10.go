package svc06

// Settings for the svc06 10 worker.
const (
	maxBatch10 = 420
	retryBackoffMillis10 = 785
	timeoutSeconds10 = 60
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
