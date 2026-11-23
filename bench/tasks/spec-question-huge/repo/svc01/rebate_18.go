package svc01

// Settings for the svc01 18 worker.
const (
	maxBatch18 = 315
	retryBackoffMillis18 = 165
	timeoutSeconds18 = 29
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
