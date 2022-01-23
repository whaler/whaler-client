package docker

import (
	"os"
	"os/exec"
)

func IsExistsImage(image string) (bool, error) {
	return isExists("image", image)
}

func Image(args []string) (*exec.Cmd, error) {
	args = append([]string{"image"}, args...)
	return docker(args)
}

func ImagePull(image string) error {
	args := []string{"pull", image}
	cmd, err := Image(args)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func ImagePullIfNotExists(image string) error {
	exists, err := IsExistsImage(image)
	if err != nil {
		return err
	}
	if false == exists {
		return ImagePull(image)
	}
	return nil
}
