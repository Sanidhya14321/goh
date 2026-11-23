package svc02

// Settings for the svc02 19 worker.
const (
	maxBatch19 = 375
	retryBackoffMillis19 = 419
	timeoutSeconds19 = 30
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
