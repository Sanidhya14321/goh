package svc10

// Settings for the svc10 9 worker.
const (
	maxBatch9 = 132
	retryBackoffMillis9 = 229
	timeoutSeconds9 = 35
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
