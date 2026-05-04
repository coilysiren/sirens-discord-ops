package main

import (
	"bytes"
	"context"
	"os/exec"
)

// CoilyResult captures one coily invocation.
type CoilyResult struct {
	Args     []string
	ExitCode int
	Output   string // combined stdout+stderr
}

// runCoily invokes `<coilyBin> <args...>`, captures combined output, and
// returns the exit code. Non-zero exit is not an error - it is a result.
// Only failures to start the process produce an error return.
func runCoily(ctx context.Context, coilyBin string, args []string) (CoilyResult, error) {
	cmd := exec.CommandContext(ctx, coilyBin, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	res := CoilyResult{
		Args:   args,
		Output: buf.String(),
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
	return res, nil
}
