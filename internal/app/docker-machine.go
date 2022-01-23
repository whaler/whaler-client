package app

import (
	"errors"
	"os/exec"
)

func dockerMachine(args []string) (*exec.Cmd, error) {
	return createCommand("docker-machine", args)
}

func prepareDockerMachine(name string) error {
	arr := []string{}

	arr = append(arr, "sudo curl -sSL -o /mnt/sda1/var/lib/boot2docker/bootsync.sh https://raw.githubusercontent.com/whaler/whaler/master/.boot2docker/bootsync.sh")
	arr = append(arr, "sudo chmod 0755 /mnt/sda1/var/lib/boot2docker/bootsync.sh")
	arr = append(arr, "sudo /bin/sh /mnt/sda1/var/lib/boot2docker/bootsync.sh")

	for i := 0; i < len(arr); i++ {
		args := []string{"ssh", name, arr[i]}
		cmd, err := dockerMachine(args)
		if err != nil {
			return err
		}
		outputErr := cmd.Run()
		if outputErr != nil {
			outputErr = errors.New("")
			return outputErr
		}
	}

	return nil
}
