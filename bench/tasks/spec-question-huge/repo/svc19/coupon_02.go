package svc19

// Settings for the svc19 2 worker.
const (
	maxBatch2 = 22
	retryBackoffMillis2 = 734
	timeoutSeconds2 = 40
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
