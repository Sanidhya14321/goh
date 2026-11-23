package svc19

// Settings for the svc19 12 worker.
const (
	maxBatch12 = 436
	retryBackoffMillis12 = 601
	timeoutSeconds12 = 44
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
