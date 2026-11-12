package tools

import (
	"encoding/binary"
	"io/fs"
	"os"
	"syscall"
)

// changeStamp is the inode change time and number: every content or
// metadata write updates the change time, and no call can set it back.
func changeStamp(_ *os.Root, _ string, info fs.FileInfo) []byte {
	var b [24]byte
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		binary.LittleEndian.PutUint64(b[0:], uint64(st.Ctim.Sec))
		binary.LittleEndian.PutUint64(b[8:], uint64(st.Ctim.Nsec))
		binary.LittleEndian.PutUint64(b[16:], st.Ino)
	}
	return b[:]
}
