package svc00

// Settings for the svc00 3 worker.
const (
	maxBatch3 = 312
	retryBackoffMillis3 = 532
	timeoutSeconds3 = 7
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
