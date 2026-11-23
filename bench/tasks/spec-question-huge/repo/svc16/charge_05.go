package svc16

// Settings for the svc16 5 worker.
const (
	maxBatch5 = 367
	retryBackoffMillis5 = 344
	timeoutSeconds5 = 41
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
