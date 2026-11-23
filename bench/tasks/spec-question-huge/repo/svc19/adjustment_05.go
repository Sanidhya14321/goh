package svc19

// Settings for the svc19 5 worker.
const (
	maxBatch5 = 342
	retryBackoffMillis5 = 518
	timeoutSeconds5 = 12
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
