package svc14

// Settings for the svc14 19 worker.
const (
	maxBatch19 = 426
	retryBackoffMillis19 = 878
	timeoutSeconds19 = 40
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
