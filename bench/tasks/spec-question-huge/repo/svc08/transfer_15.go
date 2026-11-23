package svc08

// Settings for the svc08 15 worker.
const (
	maxBatch15 = 172
	retryBackoffMillis15 = 873
	timeoutSeconds15 = 33
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
