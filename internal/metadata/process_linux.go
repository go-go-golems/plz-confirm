//go:build linux
// +build linux

package metadata

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

const maxParentDepth = 32

func collectProcessTree() (*v1.ProcessInfo, []*v1.ProcessInfo) {
	selfPID := osGetpid()
	self, ppid, err := readProcProcess(selfPID)
	if err != nil {
		// Best-effort: at least populate pid/ppid/cmdline from process environment.
		pid := int64(osGetpid())
		ppidI := int64(osGetppid())
		self = &v1.ProcessInfo{Pid: pid, Ppid: &ppidI, Argv: osArgs()}
		return self, nil
	}

	out := make([]*v1.ProcessInfo, 0, 8)
	seen := map[int]struct{}{selfPID: {}}

	curPID := ppid
	for depth := 0; depth < maxParentDepth; depth++ {
		if curPID <= 1 {
			break
		}
		if _, ok := seen[curPID]; ok {
			break
		}
		seen[curPID] = struct{}{}

		p, nextPPID, err := readProcProcess(curPID)
		if err != nil {
			break
		}
		out = append(out, p)
		curPID = nextPPID
	}

	return self, out
}

func readProcProcess(pid int) (*v1.ProcessInfo, int, error) {
	ppid, err := readProcPPID(pid)
	if err != nil {
		return nil, 0, err
	}

	comm, _ := readProcComm(pid)
	argv, _ := readProcCmdline(pid)

	pid64 := int64(pid)
	ppid64 := int64(ppid)
	pi := &v1.ProcessInfo{
		Pid:  pid64,
		Ppid: &ppid64,
	}
	if comm != "" {
		pi.Comm = &comm
	}
	if len(argv) > 0 {
		pi.Argv = argv
	}

	return pi, ppid, nil
}

func readProcComm(pid int) (string, error) {
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err != nil {
		return "", errors.Wrap(err, "read /proc/<pid>/comm")
	}
	return strings.TrimSpace(string(b)), nil
}

func readProcCmdline(pid int) ([]string, error) {
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return nil, errors.Wrap(err, "read /proc/<pid>/cmdline")
	}
	parts := bytes.Split(b, []byte{0})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		out = append(out, string(p))
	}
	return out, nil
}

func readProcPPID(pid int) (int, error) {
	f, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0, errors.Wrap(err, "open /proc/<pid>/status")
	}
	defer func() { _ = f.Close() }()

	b, err := io.ReadAll(io.LimitReader(f, 64<<10))
	if err != nil {
		return 0, errors.Wrap(err, "read /proc/<pid>/status")
	}

	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "PPid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0, errors.Wrap(err, "parse PPid")
		}
		return ppid, nil
	}

	return 0, errors.New("PPid not found in /proc/<pid>/status")
}
