package metadata

import (
	"os"
	"testing"
)

func TestCollect_SelfPID(t *testing.T) {
	md := Collect()
	if md == nil {
		t.Fatalf("Collect() returned nil")
		return
	}
	if md.Self == nil {
		t.Fatalf("Collect().Self is nil")
		return
	}
	self := md.Self
	if got, want := self.Pid, int64(os.Getpid()); got != want {
		t.Fatalf("self pid=%d want=%d", got, want)
	}
	for _, p := range md.Parents {
		if p != nil && p.Pid == self.Pid {
			t.Fatalf("parents should not contain self pid=%d", self.Pid)
		}
	}
}
