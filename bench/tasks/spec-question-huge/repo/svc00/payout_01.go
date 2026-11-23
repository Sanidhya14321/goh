package svc00

// Settings for the svc00 1 worker.
const (
	maxBatch1 = 124
	retryBackoffMillis1 = 242
	timeoutSeconds1 = 52
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
