package svc09

// Settings for the svc09 3 worker.
const (
	maxBatch3 = 149
	retryBackoffMillis3 = 443
	timeoutSeconds3 = 25
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
