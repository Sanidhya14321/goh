//go:build windows

package main

import "os"

// acpStream uses inherited handles directly on Windows. Go's runtime does not
// support nonblocking anonymous pipes there; cancellation relies on closing
// the stream instead of read deadlines.
func acpStream(file *os.File) (*os.File, func(), error) {
	return file, func() {}, nil
}
