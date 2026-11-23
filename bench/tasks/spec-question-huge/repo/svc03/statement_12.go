package svc03

// Settings for the svc03 12 worker.
const (
	maxBatch12 = 226
	retryBackoffMillis12 = 520
	timeoutSeconds12 = 34
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
