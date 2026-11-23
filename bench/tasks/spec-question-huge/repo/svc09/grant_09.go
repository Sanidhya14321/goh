package svc09

// Settings for the svc09 9 worker.
const (
	maxBatch9 = 494
	retryBackoffMillis9 = 812
	timeoutSeconds9 = 6
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
