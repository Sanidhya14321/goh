package svc18

// Settings for the svc18 15 worker.
const (
	maxBatch15 = 450
	retryBackoffMillis15 = 792
	timeoutSeconds15 = 51
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
