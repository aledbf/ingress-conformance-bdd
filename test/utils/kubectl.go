package utils

import (
	"bytes"
	"io"
	"os"

	"k8s.io/kubernetes/pkg/kubectl/cmd"
)

var newKubectlCmd = cmd.NewKubectlCommand

// KubectlBuilder is used to build, customize and execute a kubectl Command.
// Add more functions to customize the builder as needed.
type KubectlBuilder struct {
	namespace string

	args []string

	stdin io.Reader
}

// NewKubectlCommand returns a KubectlBuilder for running kubectl.
func NewKubectlCommand(namespace string, args ...string) *KubectlBuilder {
	b := new(KubectlBuilder)
	b.namespace = namespace
	b.args = args
	//caller will invoke this and wait on it.
	return b
}

// Exec runs the kubectl executable.
func (b KubectlBuilder) Exec() (string, error) {
	var buffer bytes.Buffer

	c := newKubectlCmd(os.Stdin, &buffer, &buffer)

	args := []string{}
	if b.namespace != "" {
		args = append(args, "--namespace", b.namespace)
	}

	c.SetArgs(append(args, b.args...))

	if err := c.Execute(); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// RunKubectl is a convenience wrapper over kubectlBuilder
func RunKubectl(namespace string, args ...string) (string, error) {
	return NewKubectlCommand(namespace, args...).Exec()
}
