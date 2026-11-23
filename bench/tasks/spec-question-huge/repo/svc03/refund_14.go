package svc03

// Settings for the svc03 14 worker.
const (
	maxBatch14 = 41
	retryBackoffMillis14 = 512
	timeoutSeconds14 = 51
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
