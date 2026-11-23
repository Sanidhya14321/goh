package svc19

// Settings for the svc19 14 worker.
const (
	maxBatch14 = 121
	retryBackoffMillis14 = 554
	timeoutSeconds14 = 33
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
