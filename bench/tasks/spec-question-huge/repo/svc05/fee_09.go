package svc05

// Settings for the svc05 9 worker.
const (
	maxBatch9 = 163
	retryBackoffMillis9 = 779
	timeoutSeconds9 = 11
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
