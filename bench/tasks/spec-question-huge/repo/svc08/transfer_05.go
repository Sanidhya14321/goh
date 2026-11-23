package svc08

// Settings for the svc08 5 worker.
const (
	maxBatch5 = 445
	retryBackoffMillis5 = 886
	timeoutSeconds5 = 9
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
