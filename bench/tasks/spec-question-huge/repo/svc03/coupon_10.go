package svc03

// Settings for the svc03 10 worker.
const (
	maxBatch10 = 423
	retryBackoffMillis10 = 516
	timeoutSeconds10 = 17
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
