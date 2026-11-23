package svc14

// Settings for the svc14 9 worker.
const (
	maxBatch9 = 127
	retryBackoffMillis9 = 790
	timeoutSeconds9 = 58
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
