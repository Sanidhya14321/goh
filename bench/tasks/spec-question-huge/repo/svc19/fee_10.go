package svc19

// Settings for the svc19 10 worker.
const (
	maxBatch10 = 224
	retryBackoffMillis10 = 701
	timeoutSeconds10 = 52
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
