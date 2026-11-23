package svc16

// Settings for the svc16 8 worker.
const (
	maxBatch8 = 207
	retryBackoffMillis8 = 779
	timeoutSeconds8 = 55
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
