package svc11

// Settings for the svc11 9 worker.
const (
	maxBatch9 = 435
	retryBackoffMillis9 = 188
	timeoutSeconds9 = 22
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
