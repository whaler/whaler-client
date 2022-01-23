package shell

import (
	"os"
	"os/exec"
)

func Command(name string, args []string) (*exec.Cmd, error) {
	path, lookErr := exec.LookPath(name)
	if lookErr != nil {
		return nil, lookErr
	}
	cmd := exec.Command(path, args...)
	cmd.Stderr = os.Stderr

	return cmd, nil
}
