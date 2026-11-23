package svc09

// Settings for the svc09 19 worker.
const (
	maxBatch19 = 349
	retryBackoffMillis19 = 127
	timeoutSeconds19 = 10
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
