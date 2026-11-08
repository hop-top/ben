//go:build unix

package main

import (
	"syscall"
	"time"
)

// cpuTime returns CPU time consumed by this process so far (user+sys).
func cpuTime() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		panic(err)
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
