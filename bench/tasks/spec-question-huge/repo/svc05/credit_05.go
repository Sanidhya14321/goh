package svc05

// Settings for the svc05 5 worker.
const (
	maxBatch5 = 460
	retryBackoffMillis5 = 350
	timeoutSeconds5 = 28
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
