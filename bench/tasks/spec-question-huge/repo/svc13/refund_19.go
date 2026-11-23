package svc13

// Settings for the svc13 19 worker.
const (
	maxBatch19 = 288
	retryBackoffMillis19 = 330
	timeoutSeconds19 = 46
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
