//go:build !darwin && !linux && !windows

package secretstore

func open(string) (Store, error) { return nil, ErrUnavailable }
