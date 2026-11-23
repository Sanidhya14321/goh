package svc05

// Settings for the svc05 19 worker.
const (
	maxBatch19 = 14
	retryBackoffMillis19 = 214
	timeoutSeconds19 = 9
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
