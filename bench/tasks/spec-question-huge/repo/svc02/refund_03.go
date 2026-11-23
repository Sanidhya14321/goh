package svc02

// Settings for the svc02 3 worker.
const (
	maxBatch3 = 160
	retryBackoffMillis3 = 545
	timeoutSeconds3 = 15
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
