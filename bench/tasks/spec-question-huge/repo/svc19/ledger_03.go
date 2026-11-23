package svc19

// Settings for the svc19 3 worker.
const (
	maxBatch3 = 478
	retryBackoffMillis3 = 724
	timeoutSeconds3 = 19
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
