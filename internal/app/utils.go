package app

import (
	"os"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"github.com/fatih/color"

	"whaler/client/internal/pkg/shell"
)

func convertWindowsToUnixPath(path string) string {
	path = strings.Replace(path, ":\\", "/", -1)
	path = strings.Replace(path, "\\", "/", -1)
	return "/" + path
}

func createCommand(name string, args []string) (*exec.Cmd, error) {
	return shell.Command(name, args)
}

func printErrorAndExit(err error) {
	if msg, ok := err.(*exec.ExitError); ok {
		os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
	}

	if len(err.Error()) > 0 {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Fprintf(os.Stderr, red("%v\n"), err)
	}

	os.Exit(1)
}
