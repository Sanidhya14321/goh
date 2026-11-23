package svc00

// Settings for the svc00 5 worker.
const (
	maxBatch5 = 318
	retryBackoffMillis5 = 127
	timeoutSeconds5 = 40
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
