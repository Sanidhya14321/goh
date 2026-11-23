package svc10

// Settings for the svc10 19 worker.
const (
	maxBatch19 = 417
	retryBackoffMillis19 = 679
	timeoutSeconds19 = 55
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
