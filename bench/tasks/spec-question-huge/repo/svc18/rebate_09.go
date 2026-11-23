package svc18

// Settings for the svc18 9 worker.
const (
	maxBatch9 = 99
	retryBackoffMillis9 = 305
	timeoutSeconds9 = 13
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
