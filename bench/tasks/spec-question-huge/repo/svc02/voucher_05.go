package svc02

// Settings for the svc02 5 worker.
const (
	maxBatch5 = 266
	retryBackoffMillis5 = 880
	timeoutSeconds5 = 16
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
