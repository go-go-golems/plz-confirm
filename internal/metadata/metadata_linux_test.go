//go:build linux
// +build linux

package metadata

import "testing"

func TestCollect_ParentsBestEffort(t *testing.T) {
	md := Collect()
	if md == nil || md.Self == nil {
		t.Fatalf("Collect() returned nil metadata/self")
	}

	for _, p := range md.Parents {
		if p == nil {
			continue
		}
		if p.Pid <= 0 {
			t.Fatalf("parent pid=%d, want > 0", p.Pid)
		}
		if p.Ppid != nil && *p.Ppid == p.Pid {
			t.Fatalf("parent pid=%d has ppid==pid (cycle)", p.Pid)
		}
	}
}
