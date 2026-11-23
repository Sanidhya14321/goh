package svc19

// Settings for the svc19 9 worker.
const (
	maxBatch9 = 353
	retryBackoffMillis9 = 860
	timeoutSeconds9 = 49
)

func Batch9(n int) int {
	if n > maxBatch9 {
		return maxBatch9
	}
	return n
}
