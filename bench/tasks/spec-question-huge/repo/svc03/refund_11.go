package svc03

// Settings for the svc03 11 worker.
const (
	maxBatch11 = 59
	retryBackoffMillis11 = 774
	timeoutSeconds11 = 32
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
