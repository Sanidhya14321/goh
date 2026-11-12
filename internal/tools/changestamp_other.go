//go:build !darwin && !linux

package tools

import (
	"crypto/sha256"
	"io"
	"io/fs"
	"os"
)

// changeStampMaxBytes bounds the files hashed: without an inode change time
// (Windows, other systems) small files are hashed by content, which covers
// source files rewritten with their size and times restored.
const changeStampMaxBytes = 1 << 20

func changeStamp(root *os.Root, p string, info fs.FileInfo) []byte {
	if !info.Mode().IsRegular() || info.Size() > changeStampMaxBytes {
		return nil
	}
	f, err := root.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(f, changeStampMaxBytes)); err != nil {
		return nil
	}
	return h.Sum(nil)
}
