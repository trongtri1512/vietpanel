//go:build !linux

package serverinfo

import (
	"os/exec"
	"runtime"
	"strings"
)

// platformInfo holds the uname-derived fields shown on the server info page
type platformInfo struct {
	System, Node, Release, Version, Machine, Processor string
}

// getPlatformInfo returns platform info via `uname` shell-out on non-Linux platforms (dev/stub).
func getPlatformInfo() platformInfo {
	var info platformInfo
	info.System = runtime.GOOS
	info.Machine = runtime.GOARCH
	if out, err := exec.Command("uname", "-n").Output(); err == nil {
		info.Node = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		info.Release = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("uname", "-v").Output(); err == nil {
		info.Version = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("uname", "-p").Output(); err == nil {
		info.Processor = strings.TrimSpace(string(out))
	}
	return info
}
