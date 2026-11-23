package svc10

// Settings for the svc10 17 worker.
const (
	maxBatch17 = 62
	retryBackoffMillis17 = 813
	timeoutSeconds17 = 54
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
