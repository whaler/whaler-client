package docker

import (
	"os"
	"os/exec"
)

func IsExistsContainer(container string) (bool, error) {
	return isExists("container", container)
}

func Container(args []string) (*exec.Cmd, error) {
	args = append([]string{"container"}, args...)
	return docker(args)
}

func ContainerCreate(args []string) error {
	args = append([]string{"create"}, args...)
	cmd, err := Container(args)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	return err
}

func ContainerRemove(container string) error {
	args := []string{"rm", "-fv", container}
	cmd, err := Container(args)
	if err != nil {
		return err
	}
	err = cmd.Run()
	return err
}

func ContainerKill(container string) error {
	args := []string{"kill", "--signal", "SIGHUP", container}
	cmd, err := Container(args)
	if err != nil {
		return err
	}
	cmd.Stderr = nil
	_, err = cmd.CombinedOutput()
	return err
}
