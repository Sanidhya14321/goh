package svc16

// Settings for the svc16 7 worker.
const (
	maxBatch7 = 172
	retryBackoffMillis7 = 862
	timeoutSeconds7 = 35
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
