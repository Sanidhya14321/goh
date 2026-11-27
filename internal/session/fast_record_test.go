package session

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

// strictRecord is the decoder every record went through before fastRecord.
func strictRecord(line []byte) (Record, bool) {
	var r Record
	d := json.NewDecoder(bytes.NewReader(line))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return r, false
	}
	return r, true
}

func checkFastAgrees(t *testing.T, line []byte) {
	fast, ok := fastRecord(line)
	if !ok {
		return
	}
	strict, sok := strictRecord(line)
	if !sok {
		t.Fatalf("fast path accepted what the strict decoder rejects: %q", line)
	}
	if fast.Version != strict.Version || fast.Type != strict.Type || fast.Time != strict.Time || !bytes.Equal(fast.Data, strict.Data) {
		t.Fatalf("fast %+v != strict %+v for %q", fast, strict, line)
	}
}

func TestFastRecordMatchesStrict(t *testing.T) {
	accepted := []string{
		`{"version":2,"type":"message","data":{"role":"user","content":"hi"},"time":1764200000000}`,
		`{"version":1,"type":"message","data":{"a":1,"time":5}}`,
		`{"version":2,"type":"x","data":{"k":{"time":3}},"time":9}`,
		`{"version":2,"type":"x","data":"a,\"time\":5","time":9}`,
		`{"version":2,"type":"x","data":[1,2],"time":7}`,
	}
	for _, line := range accepted {
		if _, ok := fastRecord([]byte(line)); !ok {
			t.Errorf("fast path should take %s", line)
		}
		checkFastAgrees(t, []byte(line))
	}
	declined := []string{
		`{"type":"x","version":2,"data":1,"time":3}`,
		`{"version":2,"type":"a\"b","data":1,"time":3}`,
		`{"version":2,"type":"x","data":1,"time":3,"time":4}`,
		`{"version":2,"type":"x","data":1,"time":03}`,
		`{"version":2,"type":"x","data":1,"time":3} `,
		`{"version":2,"type":"x","data":,"time":3}`,
		`{"version":2,"type":"x","data":1,"extra":2,"time":3}`,
		`{"version":20,"type":"x","data":1,"time":3}`,
	}
	for _, line := range declined {
		if _, ok := fastRecord([]byte(line)); ok {
			t.Errorf("fast path should leave %s to the strict decoder", line)
		}
	}
}

func FuzzFastRecord(f *testing.F) {
	f.Add([]byte(`{"version":2,"type":"message","data":{"a":"b"},"time":17}`))
	f.Add([]byte(`{"version":1,"type":"m","data":[{"time":1}]}`))
	f.Fuzz(func(t *testing.T, line []byte) { checkFastAgrees(t, line) })
}
