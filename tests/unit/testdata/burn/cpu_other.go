//go:build !unix

package main

import "time"

var start = time.Now()

// cpuTime falls back to wall time where getrusage is unavailable; rusage
// assertions are not exercised on these platforms.
func cpuTime() time.Duration { return time.Since(start) }
