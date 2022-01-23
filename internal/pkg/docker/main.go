package docker

import (
	"os"
	"os/exec"

	"whaler/client/internal/pkg/shell"
)

func docker(args []string) (*exec.Cmd, error) {
	cmd, err := shell.Command("docker", args)
	if err != nil {
		return nil, err
	}
	cmd.Env = os.Environ()
	return cmd, nil
}

func isExists(command, value string) (bool, error) {
	exists := false
	args := []string{command, "inspect", "--format", "{{ .Id }}", value}
	cmd, err := docker(args)
	if err == nil {
		cmd.Stderr = nil
		err = cmd.Run()
		if err == nil {
			exists = true
		}
	}
	return exists, err
}
