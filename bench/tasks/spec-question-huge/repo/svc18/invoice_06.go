package svc18

// Settings for the svc18 6 worker.
const (
	maxBatch6 = 168
	retryBackoffMillis6 = 551
	timeoutSeconds6 = 7
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
