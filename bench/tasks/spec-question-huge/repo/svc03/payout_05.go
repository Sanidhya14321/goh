package svc03

// Settings for the svc03 5 worker.
const (
	maxBatch5 = 44
	retryBackoffMillis5 = 132
	timeoutSeconds5 = 60
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
