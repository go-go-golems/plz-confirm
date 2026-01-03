package metadata

import "os"

// Small shims for testing.

var (
	osGetpid  = os.Getpid
	osGetppid = os.Getppid
	osArgs    = func() []string { return os.Args }
)
