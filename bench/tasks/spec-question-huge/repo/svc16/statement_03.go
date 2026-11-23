package svc16

// Settings for the svc16 3 worker.
const (
	maxBatch3 = 180
	retryBackoffMillis3 = 665
	timeoutSeconds3 = 53
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
