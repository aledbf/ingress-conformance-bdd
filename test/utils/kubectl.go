package utils

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	uexec "k8s.io/utils/exec"
)

// KubectlBuilder is used to build, customize and execute a kubectl Command.
// Add more functions to customize the builder as needed.
type KubectlBuilder struct {
	cmd     *exec.Cmd
	timeout <-chan time.Time
}

// NewKubectlCommand returns a KubectlBuilder for running kubectl.
func NewKubectlCommand(namespace string, args ...string) *KubectlBuilder {
	b := new(KubectlBuilder)
	b.cmd = exec.Command("kubectl")
	//caller will invoke this and wait on it.
	return b
}

// WithEnv sets the given environment and returns itself.
func (b *KubectlBuilder) WithEnv(env []string) *KubectlBuilder {
	b.cmd.Env = env
	return b
}

// WithTimeout sets the given timeout and returns itself.
func (b *KubectlBuilder) WithTimeout(t <-chan time.Time) *KubectlBuilder {
	b.timeout = t
	return b
}

// WithStdinData sets the given data to stdin and returns itself.
func (b KubectlBuilder) WithStdinData(data string) *KubectlBuilder {
	b.cmd.Stdin = strings.NewReader(data)
	return &b
}

// WithStdinReader sets the given reader and returns itself.
func (b KubectlBuilder) WithStdinReader(reader io.Reader) *KubectlBuilder {
	b.cmd.Stdin = reader
	return &b
}

// Exec runs the kubectl executable.
func (b KubectlBuilder) Exec() (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := b.cmd
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	//Logf("Running '%s %s'", cmd.Path, strings.Join(cmd.Args[1:], " ")) // skip arg[0] as it is printed separately
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting %v:\nCommand stdout:\n%v\nstderr:\n%v\nerror:\n%v",
			cmd, cmd.Stdout, cmd.Stderr, err)
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- cmd.Wait()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			var rc = 127
			if ee, ok := err.(*exec.ExitError); ok {
				rc = int(ee.Sys().(syscall.WaitStatus).ExitStatus())
			}

			return "", uexec.CodeExitError{
				Err: fmt.Errorf("error running %v:\nCommand stdout:\n%v\nstderr:\n%v\nerror:\n%v",
					cmd, cmd.Stdout, cmd.Stderr, err),
				Code: rc,
			}
		}
	case <-b.timeout:
		_ = b.cmd.Process.Kill()
		return "", fmt.Errorf("timed out waiting for command %v:\nCommand stdout:\n%v\nstderr:\n%v",
			cmd, cmd.Stdout, cmd.Stderr)
	}

	return stdout.String(), nil
}

// RunKubectl is a convenience wrapper over kubectlBuilder
func RunKubectl(namespace string, args ...string) (string, error) {
	return NewKubectlCommand(namespace, args...).Exec()
}

// RunKubectlInput is a convenience wrapper over kubectlBuilder that takes input to stdin
func RunKubectlInput(namespace string, data string, args ...string) (string, error) {
	return NewKubectlCommand(namespace, args...).WithStdinData(data).Exec()
}
