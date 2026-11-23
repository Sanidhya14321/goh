package svc10

// Settings for the svc10 6 worker.
const (
	maxBatch6 = 124
	retryBackoffMillis6 = 765
	timeoutSeconds6 = 9
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
