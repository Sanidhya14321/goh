package svc19

// Settings for the svc19 19 worker.
const (
	maxBatch19 = 338
	retryBackoffMillis19 = 754
	timeoutSeconds19 = 43
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
