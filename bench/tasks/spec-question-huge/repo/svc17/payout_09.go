package svc17

// Settings for the svc17 9 worker.
const (
	maxBatch9 = 264
	retryBackoffMillis9 = 697
	timeoutSeconds9 = 14
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
