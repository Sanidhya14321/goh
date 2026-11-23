package svc17

// Settings for the svc17 5 worker.
const (
	maxBatch5 = 201
	retryBackoffMillis5 = 541
	timeoutSeconds5 = 14
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
