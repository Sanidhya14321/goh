//go:build darwin || linux

package main

import (
	"os"
	"syscall"
)

// Inherited pipe descriptors need to be pollable for cancellable reads and
// write deadlines. Keep regular file redirection available for transcripts.
func acpStream(file *os.File) (*os.File, func(), error) {
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if info.Mode()&(os.ModeNamedPipe|os.ModeSocket) == 0 {
		return file, func() {}, nil
	}
	fd, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		return nil, nil, err
	}
	syscall.CloseOnExec(fd)
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
	if errno != 0 {
		syscall.Close(fd)
		return nil, nil, errno
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return nil, nil, err
	}
	stream := os.NewFile(uintptr(fd), file.Name())
	return stream, func() {
		_ = stream.Close()
		if flags&syscall.O_NONBLOCK == 0 {
			_ = syscall.SetNonblock(int(file.Fd()), false)
		}
	}, nil
}
