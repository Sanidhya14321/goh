package session

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNoSession means no saved session exists for the project.
var ErrNoSession = errors.New("no saved session for this project yet")

// Latest returns the most recently written session whose project is project
// (an absolute, symlink-resolved path). Damaged journals are skipped.
func Latest(dir, project string) (string, error) {
	infos, _, err := ListAll(dir)
	if err != nil {
		return "", err
	}
	best, bestTime := "", int64(0)
	for _, info := range infos {
		if info.Project != project {
			continue
		}
		st, err := os.Stat(filepath.Join(dir, info.ID+".jsonl"))
		if err != nil {
			continue
		}
		if t := st.ModTime().UnixNano(); best == "" || t > bestTime || (t == bestTime && info.ID > best) {
			best, bestTime = info.ID, t
		}
	}
	if best == "" {
		return "", ErrNoSession
	}
	return best, nil
}
