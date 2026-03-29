// Command ben-reporter-json2 is a minimal test plugin that implements the
// ben binary reporter stdio protocol. It reads {"run":{...}} from stdin
// and writes a fixed sentinel string to stdout.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	Run json.RawMessage `json:"run"`
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "ben-reporter-json2: decode:", err)
		os.Exit(1)
	}
	fmt.Print("PLUGIN_REPORTER_OK")
}
