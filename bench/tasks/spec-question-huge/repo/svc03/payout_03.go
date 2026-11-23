package svc03

// Settings for the svc03 3 worker.
const (
	maxBatch3 = 311
	retryBackoffMillis3 = 325
	timeoutSeconds3 = 5
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
