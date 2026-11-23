package svc13

// Settings for the svc13 5 worker.
const (
	maxBatch5 = 320
	retryBackoffMillis5 = 710
	timeoutSeconds5 = 55
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
