package svc07

// Settings for the svc07 9 worker.
const (
	maxBatch9 = 378
	retryBackoffMillis9 = 407
	timeoutSeconds9 = 37
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
