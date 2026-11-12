package secretstore

import (
	"bytes"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
)

// The real OS store is exercised on Windows (DPAPI needs no UI), and on
// macOS or Linux only with GOH_OS_STORE_TEST=1, since it adds an entry to
// the user's keychain (removed again before the test ends).
func TestOSStoreRoundTrip(t *testing.T) {
	if runtime.GOOS != "windows" && os.Getenv("GOH_OS_STORE_TEST") != "1" {
		t.Skip("set GOH_OS_STORE_TEST=1 to use this system's credential store")
	}
	old := service
	service = "GoHarness-test"
	t.Cleanup(func() { service = old })
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := "goh-test-entry"
	t.Cleanup(func() { _ = s.Delete(name) })
	if _, err := s.Get(name); !errors.Is(err, ErrNotFound) {
		t.Fatal("expected no entry:", err)
	}
	// Bytes a shell or a line-oriented helper would mangle.
	secret := []byte(strings.Repeat(`{"token":"a b\"c'$d\n"}`, 40) + "\x00\xff")
	for _, v := range [][]byte{secret, []byte("replaced")} {
		if err := s.Set(name, v); err != nil {
			t.Fatal(err)
		}
		got, err := s.Get(name)
		if err != nil || !bytes.Equal(got, v) {
			t.Fatalf("read back %d bytes, %v", len(got), err)
		}
	}
	if err := s.Delete(name); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(name); !errors.Is(err, ErrNotFound) {
		t.Fatal("second delete:", err)
	}
}

func TestNamesAndSizesAreChecked(t *testing.T) {
	for _, n := range []string{"", "-flag", "a b", "a\"b", "a/b", strings.Repeat("a", 200)} {
		if checkName(n) == nil {
			t.Fatalf("name %q accepted", n)
		}
	}
	if checkName("0123abcd.key") != nil || checkName("chatgpt-account.json") != nil {
		t.Fatal("valid name refused")
	}
	if checkSize(nil) == nil || checkSize(make([]byte, MaxSecret+1)) == nil {
		t.Fatal("size not checked")
	}
}
