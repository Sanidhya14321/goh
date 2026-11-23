package svc10

// Settings for the svc10 11 worker.
const (
	maxBatch11 = 402
	retryBackoffMillis11 = 477
	timeoutSeconds11 = 15
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
