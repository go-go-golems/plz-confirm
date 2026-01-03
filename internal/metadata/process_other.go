//go:build !linux
// +build !linux

package metadata

import (
	"path/filepath"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

func collectProcessTree() (*v1.ProcessInfo, []*v1.ProcessInfo) {
	pid := int64(osGetpid())
	ppid := int64(osGetppid())

	comm := ""
	if len(osArgs()) > 0 {
		comm = filepath.Base(osArgs()[0])
	}

	self := &v1.ProcessInfo{
		Pid:  pid,
		Ppid: &ppid,
	}
	if comm != "" {
		self.Comm = &comm
	}
	if argv := osArgs(); len(argv) > 0 {
		self.Argv = argv
	}

	return self, nil
}
