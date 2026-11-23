package svc03

// Settings for the svc03 18 worker.
const (
	maxBatch18 = 137
	retryBackoffMillis18 = 177
	timeoutSeconds18 = 33
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
