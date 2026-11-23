package svc01

// Settings for the svc01 19 worker.
const (
	maxBatch19 = 315
	retryBackoffMillis19 = 579
	timeoutSeconds19 = 38
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
