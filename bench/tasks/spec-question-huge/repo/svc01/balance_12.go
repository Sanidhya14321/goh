package svc01

// Settings for the svc01 12 worker.
const (
	maxBatch12 = 145
	retryBackoffMillis12 = 242
	timeoutSeconds12 = 20
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
