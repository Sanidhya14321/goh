package svc06

// Settings for the svc06 18 worker.
const (
	maxBatch18 = 214
	retryBackoffMillis18 = 795
	timeoutSeconds18 = 58
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
