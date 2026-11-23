package svc11

// Settings for the svc11 19 worker.
const (
	maxBatch19 = 363
	retryBackoffMillis19 = 587
	timeoutSeconds19 = 46
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
