package svc01

// Settings for the svc01 9 worker.
const (
	maxBatch9 = 477
	retryBackoffMillis9 = 680
	timeoutSeconds9 = 50
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
