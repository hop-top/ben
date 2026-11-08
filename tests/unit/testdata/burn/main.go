// Command burn is a test helper: it burns at least 60 ms of its OWN CPU time
// (user+sys, via getrusage), touches ~24 MB of memory, then prints its
// arguments joined by "|". Spinning on consumed CPU rather than wall time keeps
// the burn deterministic under machine load.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const burnTime = 60 * time.Millisecond

var sink []byte

func main() {
	buf := make([]byte, 24<<20)
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}
	sink = buf

	x := uint64(1)
	for cpuTime() < burnTime {
		for i := 0; i < 10000; i++ {
			x = x*6364136223846793005 + 1442695040888963407
		}
	}
	if x == 42 {
		fmt.Fprint(os.Stderr, "")
	}
	fmt.Print(strings.Join(os.Args[1:], "|"))
}
