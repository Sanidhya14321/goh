package svc04

// Settings for the svc04 19 worker.
const (
	maxBatch19 = 132
	retryBackoffMillis19 = 371
	timeoutSeconds19 = 30
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
