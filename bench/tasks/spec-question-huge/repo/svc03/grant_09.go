package svc03

// Settings for the svc03 9 worker.
const (
	maxBatch9 = 252
	retryBackoffMillis9 = 348
	timeoutSeconds9 = 55
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
