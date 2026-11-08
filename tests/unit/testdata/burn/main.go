// Command burn is a test helper: it burns ~60 ms of CPU, touches ~24 MB of
// memory, then prints its arguments joined by "|".
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var sink []byte

func main() {
	buf := make([]byte, 24<<20)
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}
	sink = buf

	deadline := time.Now().Add(60 * time.Millisecond)
	x := uint64(1)
	for time.Now().Before(deadline) {
		for i := 0; i < 10000; i++ {
			x = x*6364136223846793005 + 1442695040888963407
		}
	}
	if x == 42 {
		fmt.Fprint(os.Stderr, "")
	}
	fmt.Print(strings.Join(os.Args[1:], "|"))
}
