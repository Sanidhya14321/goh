package svc10

// Settings for the svc10 7 worker.
const (
	maxBatch7 = 396
	retryBackoffMillis7 = 131
	timeoutSeconds7 = 20
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
