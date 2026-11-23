package svc19

// Settings for the svc19 18 worker.
const (
	maxBatch18 = 495
	retryBackoffMillis18 = 565
	timeoutSeconds18 = 10
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
