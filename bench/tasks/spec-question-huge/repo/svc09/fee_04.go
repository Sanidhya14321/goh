package svc09

// Settings for the svc09 4 worker.
const (
	maxBatch4 = 51
	retryBackoffMillis4 = 241
	timeoutSeconds4 = 14
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
