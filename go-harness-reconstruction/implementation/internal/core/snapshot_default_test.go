package core

import "testing"

func TestSnapshotDefaults(t *testing.T) {
	for raw, want := range map[string][2]bool{
		`{}`:                   {true, false},
		`{"snapshots": true}`:  {true, true},
		`{"snapshots": false}`: {false, false},
	} {
		c, err := DecodeConfig(raw)
		if err != nil {
			t.Fatal(err)
		}
		if c.Snapshots != want[0] || c.SnapshotsStrict != want[1] {
			t.Fatalf("%s: snapshots=%v strict=%v", raw, c.Snapshots, c.SnapshotsStrict)
		}
	}
	if Defaults().Snapshots {
		t.Fatal("Defaults() itself stays off; loaders turn checkpoints on")
	}
}
