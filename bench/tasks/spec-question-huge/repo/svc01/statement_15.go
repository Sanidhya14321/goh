package svc01

// Settings for the svc01 15 worker.
const (
	maxBatch15 = 122
	retryBackoffMillis15 = 241
	timeoutSeconds15 = 37
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
