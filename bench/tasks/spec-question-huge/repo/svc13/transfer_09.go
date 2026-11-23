package svc13

// Settings for the svc13 9 worker.
const (
	maxBatch9 = 40
	retryBackoffMillis9 = 724
	timeoutSeconds9 = 52
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
