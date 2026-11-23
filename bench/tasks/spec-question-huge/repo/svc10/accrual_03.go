package svc10

// Settings for the svc10 3 worker.
const (
	maxBatch3 = 152
	retryBackoffMillis3 = 870
	timeoutSeconds3 = 58
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
