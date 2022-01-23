package docker

import (
	"os/exec"
)

func IsExistsNetwork(network string) (bool, error) {
	return isExists("network", network)
}

func Network(args []string) (*exec.Cmd, error) {
	args = append([]string{"network"}, args...)
	return docker(args)
}

func NetworkCreate(network string) error {
	args := []string{"create", network}
	cmd, err := Network(args)
	if err != nil {
		return err
	}
	cmd.Stderr = nil
	err = cmd.Run()
	return err
}
