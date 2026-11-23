package svc04

// Settings for the svc04 5 worker.
const (
	maxBatch5 = 366
	retryBackoffMillis5 = 848
	timeoutSeconds5 = 55
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
