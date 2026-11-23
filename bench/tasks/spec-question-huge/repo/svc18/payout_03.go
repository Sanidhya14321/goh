package svc18

// Settings for the svc18 3 worker.
const (
	maxBatch3 = 320
	retryBackoffMillis3 = 460
	timeoutSeconds3 = 19
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
