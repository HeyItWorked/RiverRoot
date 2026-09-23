package runner

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// exitCodeFromErr splits an exec result into an exit code and a real error.
func exitCodeFromErr(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 0, err
}

// RunCommand runs cmd with args in workdir, kills it after timeout.
func RunCommand(cmd string, args []string, workdir string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = workdir

	var outBuf, errBuf bytes.Buffer
	c.Stdout = &outBuf
	c.Stderr = &errBuf

	runErr := c.Run()
	exitCode, err = exitCodeFromErr(runErr)
	if err != nil {
		return
	}

	stdout = outBuf.String()
	stderr = errBuf.String()

	return
}
