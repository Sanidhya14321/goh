package svc06

// Settings for the svc06 3 worker.
const (
	maxBatch3 = 470
	retryBackoffMillis3 = 140
	timeoutSeconds3 = 27
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
