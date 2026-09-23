// Package runner executes pipeline commands, either as local processes or
// inside Docker containers, with each run bounded by a timeout.
package runner

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// RunInContainer runs cmd inside a Docker container using the Docker Go SDK.
// Mounts workdir to /workspace inside the container and sets it as the working directory.
func RunInContainer(img, workdir string, cmd []string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// connect to the Docker daemon using env vars (DOCKER_HOST etc.) — same as the CLI does
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return
	}

	// pull the image — returns a reader that must be drained or the pull won't complete
	reader, err := cli.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return
	}
	discardCloser(reader)

	// create the container — Config is the process config, HostConfig is the host/mount config
	// bind mount maps workdir on the host to /workspace inside the container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      img,
		Cmd:        cmd,
		WorkingDir: "/workspace",
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: workdir,
				Target: "/workspace",
			},
		},
	}, nil, nil, "")
	if err != nil {
		return
	}

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return
	}

	// stream logs with Follow:true — blocks until the container exits and the stream closes
	// ContainerLogs returns a multiplexed stream with both stdout and stderr interleaved
	logs, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		return
	}

	// stdcopy.StdCopy decodes the multiplexed stream and writes each to the correct buffer
	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logs); err != nil {
		return "", "", 0, err
	}
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	// wait for the container to exit — returns two channels, not a direct error
	// select picks whichever channel fires first
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		err = waitErr
		return
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	}

	// remove the container so stopped containers don't accumulate
	err = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
	return
}

// discardCloser drains and closes the reader — required for ImagePull to complete
func discardCloser(r io.ReadCloser) {
	// Copy errors are safe to ignore here — we're just draining the pull output.
	_, _ = io.Copy(io.Discard, r)
	r.Close()
}
