package svc01

// Settings for the svc01 5 worker.
const (
	maxBatch5 = 204
	retryBackoffMillis5 = 376
	timeoutSeconds5 = 45
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
