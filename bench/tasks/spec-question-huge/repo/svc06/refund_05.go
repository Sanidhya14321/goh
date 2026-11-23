package svc06

// Settings for the svc06 5 worker.
const (
	maxBatch5 = 191
	retryBackoffMillis5 = 898
	timeoutSeconds5 = 40
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
