package svc04

// Settings for the svc04 9 worker.
const (
	maxBatch9 = 392
	retryBackoffMillis9 = 421
	timeoutSeconds9 = 8
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
