package svc06

// Settings for the svc06 9 worker.
const (
	maxBatch9 = 387
	retryBackoffMillis9 = 440
	timeoutSeconds9 = 55
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
