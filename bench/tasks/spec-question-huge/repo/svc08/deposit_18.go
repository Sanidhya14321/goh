package svc08

// Settings for the svc08 18 worker.
const (
	maxBatch18 = 390
	retryBackoffMillis18 = 586
	timeoutSeconds18 = 33
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
