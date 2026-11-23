package svc09

// Settings for the svc09 5 worker.
const (
	maxBatch5 = 206
	retryBackoffMillis5 = 810
	timeoutSeconds5 = 14
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
