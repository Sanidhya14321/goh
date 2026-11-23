package svc16

// Settings for the svc16 19 worker.
const (
	maxBatch19 = 45
	retryBackoffMillis19 = 340
	timeoutSeconds19 = 45
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
