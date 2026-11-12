package secretstore

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// dpapi keeps each entry as a file encrypted with the user's DPAPI key, so
// another account (or a copy of the file on another machine) cannot read it.
type dpapi struct{ dir string }

func open(dir string) (Store, error) {
	if dir == "" {
		return nil, ErrUnavailable
	}
	return dpapi{dir}, nil
}

func (dpapi) Name() string { return "Windows DPAPI" }

func (d dpapi) path(name string) string { return filepath.Join(d.dir, name+".dpapi") }

func blob(b []byte) *windows.DataBlob {
	if len(b) == 0 {
		return &windows.DataBlob{}
	}
	return &windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}

func entropy() *windows.DataBlob { return blob([]byte(Service)) }

func (d dpapi) Get(name string) ([]byte, error) {
	if err := checkName(name); err != nil {
		return nil, err
	}
	f, err := os.Open(d.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sealed, err := io.ReadAll(io.LimitReader(f, 4*MaxSecret))
	if err != nil {
		return nil, err
	}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(blob(sealed), nil, entropy(), 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.New("Windows could not decrypt the saved credential for this account")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func (d dpapi) Set(name string, data []byte) error {
	if err := checkName(name); err != nil {
		return err
	}
	if err := checkSize(data); err != nil {
		return err
	}
	var out windows.DataBlob
	if err := windows.CryptProtectData(blob(data), nil, entropy(), 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return err
	}
	sealed := append([]byte(nil), unsafe.Slice(out.Data, out.Size)...)
	windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if err := os.MkdirAll(d.dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(d.dir, ".goh-dpapi-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(sealed)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), d.path(name))
}

func (d dpapi) Delete(name string) error {
	if err := checkName(name); err != nil {
		return err
	}
	err := os.Remove(d.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	return err
}
