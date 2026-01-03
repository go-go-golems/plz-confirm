package metadata

import (
	"os"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

func Collect() *v1.RequestMetadata {
	md := &v1.RequestMetadata{}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		md.Cwd = &cwd
	}

	self, parents := collectProcessTree()
	if self != nil {
		md.Self = self
	}
	if len(parents) > 0 {
		md.Parents = parents
	}

	if md.Cwd == nil && md.Self == nil && len(md.Parents) == 0 {
		return nil
	}

	return md
}
