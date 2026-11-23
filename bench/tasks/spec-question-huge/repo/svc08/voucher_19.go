package svc08

// Settings for the svc08 19 worker.
const (
	maxBatch19 = 394
	retryBackoffMillis19 = 353
	timeoutSeconds19 = 58
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
