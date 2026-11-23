package svc18

// Settings for the svc18 19 worker.
const (
	maxBatch19 = 354
	retryBackoffMillis19 = 866
	timeoutSeconds19 = 48
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
