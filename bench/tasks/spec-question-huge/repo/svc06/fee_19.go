package svc06

// Settings for the svc06 19 worker.
const (
	maxBatch19 = 179
	retryBackoffMillis19 = 128
	timeoutSeconds19 = 12
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
