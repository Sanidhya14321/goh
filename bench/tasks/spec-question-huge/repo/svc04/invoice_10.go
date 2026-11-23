package svc04

// Settings for the svc04 10 worker.
const (
	maxBatch10 = 309
	retryBackoffMillis10 = 588
	timeoutSeconds10 = 37
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
