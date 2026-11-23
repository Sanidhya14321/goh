package svc15

// Settings for the svc15 9 worker.
const (
	maxBatch9 = 24
	retryBackoffMillis9 = 745
	timeoutSeconds9 = 43
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
