package svc09

// Settings for the svc09 2 worker.
const (
	maxBatch2 = 49
	retryBackoffMillis2 = 830
	timeoutSeconds2 = 23
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
