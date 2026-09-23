package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
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

// RunStreaming runs cmd in workdir and writes each output line to out.
func RunStreaming(cmd string, args []string, workdir string, timeout time.Duration, out io.Writer) (exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = workdir

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return
	}
	stderrPipe, err := c.StderrPipe()
	if err != nil {
		return
	}

	err = c.Start()
	if err != nil {
		return
	}

	var wg sync.WaitGroup

	scanPipe := func(pipe io.ReadCloser) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			fmt.Fprintf(out, "[%s] %s\n", time.Now().Format("15:04:05"), scanner.Text())
		}
	}

	wg.Add(2)
	go scanPipe(stdoutPipe)
	go scanPipe(stderrPipe)

	wg.Wait()
	exitCode, err = exitCodeFromErr(c.Wait())

	return
}
