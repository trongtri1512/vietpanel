//go:build linux

package serverinfo

import (
	"os/exec"
	"strings"
	"syscall"
)

// platformInfo holds the uname-derived fields shown on the server info page
type platformInfo struct {
	System, Node, Release, Version, Machine, Processor string
}

func unameToString(b [65]int8) string {
	buf := make([]byte, 0, len(b))
	for _, c := range b {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}
	return string(buf)
}

// getPlatformInfo reads system/node/release/version/machine via syscall.Uname (no subprocess needed), and shells out to `uname -p` for the processor field, which has no equivalent in the uname struct
func getPlatformInfo() platformInfo {
	var info platformInfo
	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err == nil {
		info.System = unameToString(uts.Sysname)
		info.Node = unameToString(uts.Nodename)
		info.Release = unameToString(uts.Release)
		info.Version = unameToString(uts.Version)
		info.Machine = unameToString(uts.Machine)
	}
	if out, err := exec.Command("uname", "-p").Output(); err == nil {
		info.Processor = strings.TrimSpace(string(out))
	}
	return info
}
