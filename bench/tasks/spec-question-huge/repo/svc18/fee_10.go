package svc18

// Settings for the svc18 10 worker.
const (
	maxBatch10 = 458
	retryBackoffMillis10 = 474
	timeoutSeconds10 = 38
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
