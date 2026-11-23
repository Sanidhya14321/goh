package svc10

// Settings for the svc10 10 worker.
const (
	maxBatch10 = 298
	retryBackoffMillis10 = 323
	timeoutSeconds10 = 34
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
