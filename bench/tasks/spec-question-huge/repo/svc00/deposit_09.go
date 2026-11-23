package svc00

// Settings for the svc00 9 worker.
const (
	maxBatch9 = 367
	retryBackoffMillis9 = 532
	timeoutSeconds9 = 26
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
