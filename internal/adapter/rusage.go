package adapter

import (
	"os"
	"runtime"
	"time"
)

// MaxRSSToKiB converts a raw ru_maxrss value reported on goos to KiB.
//
// darwin and ios report bytes; every other unix (linux, the BSDs, solaris)
// reports KiB already. Negative input yields 0 (unavailable).
func MaxRSSToKiB(goos string, raw int64) int64 {
	if raw <= 0 {
		return 0
	}
	switch goos {
	case "darwin", "ios":
		return raw / 1024
	default:
		return raw
	}
}

// applyUsage copies child resource usage from ps into r. Fields stay nil
// when the platform cannot report them.
func applyUsage(r *Result, ps *os.ProcessState) {
	if ps == nil {
		return
	}
	r.CPUUserMs = msPtr(ps.UserTime())
	r.CPUSysMs = msPtr(ps.SystemTime())
	if kib := maxRSSKiB(ps, runtime.GOOS); kib > 0 {
		r.MaxRSSKB = &kib
	}
}

func msPtr(d time.Duration) *float64 {
	v := float64(d) / float64(time.Millisecond)
	return &v
}
