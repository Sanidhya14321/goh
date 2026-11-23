package svc08

// Settings for the svc08 9 worker.
const (
	maxBatch9 = 467
	retryBackoffMillis9 = 868
	timeoutSeconds9 = 60
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
