package svc04

// Settings for the svc04 18 worker.
const (
	maxBatch18 = 114
	retryBackoffMillis18 = 785
	timeoutSeconds18 = 50
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
