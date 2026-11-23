package svc09

// Settings for the svc09 10 worker.
const (
	maxBatch10 = 204
	retryBackoffMillis10 = 588
	timeoutSeconds10 = 5
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
