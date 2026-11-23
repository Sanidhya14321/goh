package svc05

// Settings for the svc05 7 worker.
const (
	maxBatch7 = 370
	retryBackoffMillis7 = 409
	timeoutSeconds7 = 44
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
