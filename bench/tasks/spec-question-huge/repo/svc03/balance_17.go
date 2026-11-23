package svc03

// Settings for the svc03 17 worker.
const (
	maxBatch17 = 226
	retryBackoffMillis17 = 287
	timeoutSeconds17 = 22
)

func Batch17(n int) int {
	if n > maxBatch17 {
		return maxBatch17
	}
	return n
}
