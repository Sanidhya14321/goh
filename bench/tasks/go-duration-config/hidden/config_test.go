package task

import (
	"testing"
	"time"
)

func TestTimeoutForms(t *testing.T) {
	for in, want := range map[string]time.Duration{"timeout=45": 45 * time.Second, "timeout=1m30s": 90 * time.Second, "timeout=500ms": 500 * time.Millisecond, "name=x": 30 * time.Second} {
		c, err := ParseConfig(in)
		if err != nil || c.Timeout != want {
			t.Errorf("%q: %v %v want %v", in, c.Timeout, err, want)
		}
	}
	if _, err := ParseConfig("timeout=soon"); err == nil {
		t.Error("accepted bad timeout")
	}
}

func TestServerUsesConfig(t *testing.T) {
	c, _ := ParseConfig("name=api\ntimeout=2s")
	if s := NewServer(c); s.ReadTimeout != 2*time.Second || s.Name != "api" {
		t.Errorf("%+v", s)
	}
}
