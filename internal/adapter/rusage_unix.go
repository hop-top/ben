//go:build unix

package adapter

import (
	"os"
	"syscall"
)

// maxRSSKiB returns the peak RSS of the waited child in KiB, or 0 when
// unavailable.
func maxRSSKiB(ps *os.ProcessState, goos string) int64 {
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok || ru == nil {
		return 0
	}
	return MaxRSSToKiB(goos, int64(ru.Maxrss))
}
