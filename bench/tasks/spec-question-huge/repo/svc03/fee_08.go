package svc03

// Settings for the svc03 8 worker.
const (
	maxBatch8 = 77
	retryBackoffMillis8 = 840
	timeoutSeconds8 = 41
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
