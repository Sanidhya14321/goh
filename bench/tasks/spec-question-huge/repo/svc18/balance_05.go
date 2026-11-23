package svc18

// Settings for the svc18 5 worker.
const (
	maxBatch5 = 331
	retryBackoffMillis5 = 199
	timeoutSeconds5 = 45
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
