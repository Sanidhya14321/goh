package svc10

// Settings for the svc10 5 worker.
const (
	maxBatch5 = 393
	retryBackoffMillis5 = 652
	timeoutSeconds5 = 8
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
