package svc10

// Settings for the svc10 2 worker.
const (
	maxBatch2 = 242
	retryBackoffMillis2 = 434
	timeoutSeconds2 = 26
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
