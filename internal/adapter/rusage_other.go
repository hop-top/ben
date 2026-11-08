//go:build !unix

package adapter

import "os"

// maxRSSKiB is unavailable on this platform (e.g. windows); the metric is
// omitted rather than reported as 0.
func maxRSSKiB(*os.ProcessState, string) int64 { return 0 }
