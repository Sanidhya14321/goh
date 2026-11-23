package svc18

// Settings for the svc18 7 worker.
const (
	maxBatch7 = 196
	retryBackoffMillis7 = 849
	timeoutSeconds7 = 13
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
