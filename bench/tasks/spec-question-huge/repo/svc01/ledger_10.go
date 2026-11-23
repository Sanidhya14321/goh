package svc01

// Settings for the svc01 10 worker.
const (
	maxBatch10 = 118
	retryBackoffMillis10 = 771
	timeoutSeconds10 = 36
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
