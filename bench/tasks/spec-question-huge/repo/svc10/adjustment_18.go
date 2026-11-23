package svc10

// Settings for the svc10 18 worker.
const (
	maxBatch18 = 445
	retryBackoffMillis18 = 800
	timeoutSeconds18 = 43
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
