package svc10

// Settings for the svc10 12 worker.
const (
	maxBatch12 = 320
	retryBackoffMillis12 = 865
	timeoutSeconds12 = 50
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
