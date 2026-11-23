package svc12

// Settings for the svc12 9 worker.
const (
	maxBatch9 = 108
	retryBackoffMillis9 = 321
	timeoutSeconds9 = 52
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
