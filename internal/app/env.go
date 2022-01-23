package app

import (
	"os"
	"net"
	"errors"
	"regexp"
	"runtime"
	"strings"
)

const NODE_VERSION = "14.18.3"

func prepareAppEnv() error {
	if os.Getenv("WHALER_NODE_VERSION") == "" {
		os.Setenv("WHALER_NODE_VERSION", NODE_VERSION)
	}

	if os.Getenv("WHALER_BIN") == "" {
		os.Setenv("WHALER_BIN", "whaler")
	}

	if runtime.GOOS == "windows" {
		PWD, _ := os.Getwd()
		os.Setenv("PWD", convertWindowsToUnixPath(PWD))
		os.Setenv("HOME", convertWindowsToUnixPath(os.Getenv("USERPROFILE")))
	}

	if os.Getenv("PWD") == "" {
		return errors.New("\nRequired `PWD` enviroment variable are missing.\n")
	}

	if os.Getenv("HOME") == "" {
		return errors.New("\nRequired `HOME` enviroment variable are missing.\n")
	}

	if os.Getenv("WHALER_HOME") == "" {
		os.Setenv("WHALER_HOME", os.Getenv("HOME"))
	}

	if os.Getenv("DOCKER_MACHINE_NAME") != "" {
		args := []string{"env", os.Getenv("DOCKER_MACHINE_NAME")}
		cmd, err := dockerMachine(args)
		if err != nil {
			return err
		}
		out, err := cmd.Output()
		if err != nil {
			return err
		}
		r, _ := regexp.Compile("export ([A-Z_]+)=\"(.+)\"")
		env := r.FindAllStringSubmatch(string(out), -1)
		for line := 0; line < len(env); line++ {
			os.Setenv(env[line][1], env[line][2])
			if env[line][1] == "DOCKER_HOST" {
				r, _ := regexp.Compile("tcp://([0-9.]+):([0-9]+)")
				os.Setenv("DOCKER_IP", r.FindStringSubmatch(env[line][2])[1])
			}
		}

	} else {
		docker0, err := net.InterfaceByName("docker0")
		if err == nil {
			addrs, err := docker0.Addrs()
			if err == nil && len(addrs) > 0 {
				tcpAddr := &net.TCPAddr {
					IP: addrs[0].(*net.IPNet).IP,
				}
				os.Setenv("DOCKER_IP", strings.Split(tcpAddr.String(), ":")[0])
			}
		}
	}

	return nil
}
