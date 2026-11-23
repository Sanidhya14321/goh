package svc04

// Settings for the svc04 14 worker.
const (
	maxBatch14 = 216
	retryBackoffMillis14 = 222
	timeoutSeconds14 = 41
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
