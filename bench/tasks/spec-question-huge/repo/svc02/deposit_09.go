package svc02

// Settings for the svc02 9 worker.
const (
	maxBatch9 = 286
	retryBackoffMillis9 = 897
	timeoutSeconds9 = 38
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
