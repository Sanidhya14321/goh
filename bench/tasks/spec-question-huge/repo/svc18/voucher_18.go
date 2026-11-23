package svc18

// Settings for the svc18 18 worker.
const (
	maxBatch18 = 284
	retryBackoffMillis18 = 226
	timeoutSeconds18 = 34
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
