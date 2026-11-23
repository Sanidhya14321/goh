package svc00

// Settings for the svc00 19 worker.
const (
	maxBatch19 = 453
	retryBackoffMillis19 = 203
	timeoutSeconds19 = 29
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
