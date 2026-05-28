package bot

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

// runCoily runs `<coilyBin> <args...>` and captures combined output.
// Non-zero exit is a result, not an error. Only start failures return an error.
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
