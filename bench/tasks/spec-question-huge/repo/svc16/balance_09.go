package svc16

// Settings for the svc16 9 worker.
const (
	maxBatch9 = 263
	retryBackoffMillis9 = 137
	timeoutSeconds9 = 13
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
