package svc03

// Settings for the svc03 19 worker.
const (
	maxBatch19 = 60
	retryBackoffMillis19 = 151
	timeoutSeconds19 = 46
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
