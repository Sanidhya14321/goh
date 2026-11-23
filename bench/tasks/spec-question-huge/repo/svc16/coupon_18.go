package svc16

// Settings for the svc16 18 worker.
const (
	maxBatch18 = 456
	retryBackoffMillis18 = 654
	timeoutSeconds18 = 7
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
