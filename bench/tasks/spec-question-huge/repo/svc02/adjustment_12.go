package svc02

// Settings for the svc02 12 worker.
const (
	maxBatch12 = 132
	retryBackoffMillis12 = 159
	timeoutSeconds12 = 20
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
