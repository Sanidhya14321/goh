package svc17

// Settings for the svc17 18 worker.
const (
	maxBatch18 = 193
	retryBackoffMillis18 = 688
	timeoutSeconds18 = 23
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
