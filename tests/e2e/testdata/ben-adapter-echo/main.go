// Command ben-adapter-echo is a minimal test plugin that implements the
// ben binary adapter stdio protocol. It reads the request from stdin and
// unconditionally returns a fixed response.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	Action string `json:"action"`
}

type response struct {
	Metrics map[string]float64 `json:"metrics"`
	Output  string             `json:"output"`
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "ben-adapter-echo: decode:", err)
		os.Exit(1)
	}
	resp := response{
		Metrics: map[string]float64{
			"latency_ms":  1,
			"items_count": 7,
		},
		Output: "echo-plugin-output",
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fmt.Fprintln(os.Stderr, "ben-adapter-echo: encode:", err)
		os.Exit(1)
	}
}
