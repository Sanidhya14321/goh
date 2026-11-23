package svc13

// Settings for the svc13 10 worker.
const (
	maxBatch10 = 495
	retryBackoffMillis10 = 881
	timeoutSeconds10 = 18
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
