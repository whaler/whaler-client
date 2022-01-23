package app

import (
	"os"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"strconv"
	"strings"
	"io/ioutil"
	"os/signal"
	"github.com/fatih/flags"
	"github.com/Jeffail/gabs/v2"
	"golang.org/x/crypto/ssh/terminal"

	"whaler/client/internal/pkg/docker"
)

// Return cursor to start of line and clean it
const RESET_LINE = "\r\033[K\r"

func Run(appVersion string) {
	if len(os.Args[1:]) > 0 && os.Args[1] == "version" {
		fmt.Printf("%v\n", appVersion)
		os.Exit(0)
	}

	var err interface{Error() string} = nil

	nodeVersion := os.Getenv("WHALER_NODE_VERSION")

	err = prepareAppEnv()

	if err == nil {
		if len(os.Args[1:]) > 0 && os.Args[1] == "self-update" {
			_, permissionsErr := makeSelfUpdate()
			if permissionsErr != nil {
				if runtime.GOOS == "windows" {
					printErrorAndExit(permissionsErr)
				}
				sudoSelfUpdate()
			}
			os.Exit(0)
		}

		version := ""
		if len(os.Args[1:]) > 0 && os.Args[1] == "setup" {
			arr := os.Args[1:]
			if flags.Has("--version", arr) {
				val, _ := flags.Value("--version", arr)
				if val != "" {
					version = val
				}
			}
			if version == "" {
				version = "latest"
			}
			os.Args = os.Args[0:1]
		}

		doSetup := false

		_, checkErr := docker.IsExistsContainer("whaler")
		if checkErr != nil {
			if version == "" {
				version = "latest"
			}
			doSetup = true

		} else if version != "" {
			doSetup = true
		}

		if doSetup {
			updated, permissionsErr := trySelfUpdate(appVersion)

			if permissionsErr != nil {
				if runtime.GOOS == "windows" {
					printErrorAndExit(permissionsErr)
				}

				sudoSelfUpdate()
				updated = true
			}

			if updated {
				if selfPath, pathErr := getSelfPath(); pathErr == nil {
					args := os.Args[1:]
					if len(args) == 0 && version != "" {
						args = append(args, "setup", "--version", version)
					}
					if cmd, cmdErr := createCommand(selfPath, args); cmdErr == nil {
						if checkErr == nil {
							removeAppContainer()
						}

						os.Setenv("WHALER_NODE_VERSION", nodeVersion)

						cmd.Env = os.Environ()
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						err = cmd.Run()

						if err != nil {
							if msg, ok := err.(*exec.ExitError); ok {
								os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
							}
						}
						os.Exit(0)
					}
				}
			}

			if checkErr != nil {
				err = createAppContainer()
			} else {
				if nodeVersion, errVersion := getAppNodeVersion(); errVersion == nil {
					if os.Getenv("WHALER_NODE_VERSION") != nodeVersion {
						err = removeAppContainer()
						if err == nil {
							err = createAppContainer()
						}
					}
				}
			}

			if err == nil {
				err = setupApp(version)
			}
		}

		if err == nil {
			err = runApp()
		}
	}

	if err != nil {
		printErrorAndExit(err)
	}
}

func getAppFrontend() string {
	frontend := "noninteractive"

	if os.Getenv("WHALER_FRONTEND") != "" {
		frontend = os.Getenv("WHALER_FRONTEND")
	} else if terminal.IsTerminal(int(os.Stdin.Fd())) {
		frontend = "interactive"
	}

	return frontend
}

func getAppNodeVersion() (string, error) {
	docker.ImagePullIfNotExists("node:" + os.Getenv("WHALER_NODE_VERSION"))

	args := []string{"run", "-t", "--rm", "--entrypoint=node",
		"--volumes-from", "whaler",
		"node:" + os.Getenv("WHALER_NODE_VERSION"),
		"-v"}
	cmd, err := docker.Container(args)
	out := ""
	if err == nil {
		cmd.Stderr = nil
		var result []byte
		result, err = cmd.Output()

		if err == nil {
			out = strings.TrimSpace(string(result[1:]))
		}
	}

	return out, err
}

func createWhalerDir(path string) error {
	args := []string{"run", "--rm",
		"-v", path + ":/.whaler_tmp",
		"node:" + os.Getenv("WHALER_NODE_VERSION"),
		"mkdir", "-p", "/.whaler_tmp/whaler"}

	cmd, err := docker.Container(args)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func createAppContainer() error {
	if os.Getenv("DOCKER_MACHINE_NAME") != "" {
		err := prepareDockerMachine(os.Getenv("DOCKER_MACHINE_NAME"))
		if err != nil {
			return err
		}
	}

	docker.ImagePullIfNotExists("node:" + os.Getenv("WHALER_NODE_VERSION"))

	etcWhaler := "/etc/whaler:/etc/whaler"
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		etcWhaler = "/etc/whaler"
	} else {
		createWhalerDir("/etc")
	}

	createWhalerDir("/var/lib")

	args := []string{
		"--name", "whaler",
		"-v", "/.npm",
		"-v", "/.node-gyp",
		"-v", "/usr/local/bin",
		"-v", "/usr/local/lib/node_modules",
		"-v", etcWhaler,
		"-v", "/var/lib/whaler:/var/lib/whaler",
		"-v", "/var/lib/docker:/var/lib/docker",
		"-v", "/var/run/docker.sock:/var/run/docker.sock"}

	if os.Getenv("DOCKER_MACHINE_NAME") != "" {
		args = append(args, "-v", "/mnt/sda1:/mnt/sda1")
	}

	args = append(args, "node:" + os.Getenv("WHALER_NODE_VERSION"))

	return docker.ContainerCreate(args)
}

func removeAppContainer() error {
	return docker.ContainerRemove("whaler")
}

func setupApp(version string) error {
	args := []string{"run", "--name", "whaler_setup", "-t", "--rm",
		"--volumes-from", "whaler"}

	args = append(args, "-e", "NPM_CONFIG_CACHE=/.npm")
	args = append(args, "-e", "NPM_CONFIG_DEVDIR=/.node-gyp")

	if version == "dev" {
		args = append(args, "-e", "WHALER_SETUP=dev", "node:" + os.Getenv("WHALER_NODE_VERSION"))
		args = append(args, "npm", "install", "https://github.com/whaler/whaler.git")
	} else {
		args = append(args, "node:" + os.Getenv("WHALER_NODE_VERSION"))
		args = append(args, "npm", "install", "whaler@" + version)
	}
	args = append(args, "--global", "--production", "--unsafe-perm", "--no-update-notifier")

	cmd, err := docker.Container(args)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	err = cmd.Run()

	return err
}

func runApp() error {
	args := []string{"run",
		"--pid", "host",
		"--volumes-from", "whaler",
		"-e", "NPM_CONFIG_CACHE=/.npm",
		"-e", "NPM_CONFIG_DEVDIR=/.node-gyp",
		"-e", "HOME=" + os.Getenv("WHALER_HOME"),
		"-v", os.Getenv("HOME") + ":" + os.Getenv("HOME"),
		"-v", os.Getenv("HOME") + "/.whaler" + ":" + os.Getenv("WHALER_HOME") + "/.whaler"}

	content, readErr := ioutil.ReadFile(os.Getenv("HOME") + "/.whaler/client.json")
	if readErr == nil {
		parsed, parseErr := gabs.ParseJSON(content)
		if parseErr != nil {
			return parseErr
		}

		children := parsed.S("file-sharing").Children()
		for _, child := range children {
			sharedVol := child.Data().(string)
			if runtime.GOOS == "windows" {
				sharedVol = convertWindowsToUnixPath(sharedVol)
			}
			args = append(args, "-v", sharedVol + ":" + sharedVol)
		}
	}

	if os.Getenv("WHALER_PATH") != "" {
		args = append(args, "-v", os.Getenv("WHALER_PATH") + ":" + "/usr/local/lib/node_modules/whaler")
	}

	if os.Getenv("WHALER_APP") != "" {
		args = append(args, "-e", "WHALER_APP=" + os.Getenv("WHALER_APP"))
	}

	if os.Getenv("DOCKER_IP") != "" {
		args = append(args, "-e", "WHALER_DOCKER_IP=" + os.Getenv("DOCKER_IP"))
	}

	daemon := ""
	detach := ""

	if len(os.Args[1:]) > 0 && os.Args[1] == "daemon" {
		arr := os.Args[1:]
		if !(flags.Has("-h", arr) || flags.Has("--help", arr)) {
			if flags.Has("--port", arr) {
				val, _ := flags.Value("--port", arr)
				if val != "" {
					daemon = val
				}
			} else {
				daemon = "1337"
			}
		}
	} else if os.Getenv("WHALER_DETACH") != "" {
		detach = os.Getenv("WHALER_DETACH")
	}

	containerName := ""
	if daemon != "" || detach != "" {
		args = append(args, "-e", "WHALER_DAEMON_APPS=" + os.Getenv("HOME") + "/apps")
		args = append(args, "-v", os.Getenv("HOME") + "/apps" + ":" + os.Getenv("WHALER_HOME") + "/apps")

		args = append(args, "-w", os.Getenv("WHALER_HOME") + "/apps")
		args = append(args, "-d", "--restart", "always")

		if detach != "" {
			args = append(args, "--name", detach)

		} else {
			args = append(args, "--name", "whaler_daemon_" + strconv.Itoa(syscall.Getpid()))
			args = append(args, "-p", daemon + ":" + daemon)
		}

	} else {
		containerName = "whaler_" + strconv.Itoa(syscall.Getpid())
		args = append(args, "-w", os.Getenv("PWD"))
		args = append(args, "--name", containerName)

		frontend := getAppFrontend()
		args = append(args, "-e", "WHALER_FRONTEND=" + frontend)

		if frontend == "interactive" {
			args = append(args, "-it")
		}
		args = append(args, "--rm")
	}

	networkExists, err := docker.IsExistsNetwork("whaler_nw")
	if err == nil {
		if networkExists != true {
			if err := docker.NetworkCreate("whaler_nw"); err == nil {
				networkExists = true
			}
		}
	}
	if networkExists == true {
		args = append(args, "--network", "whaler_nw")
	}

	args = append(args, "node:" + os.Getenv("WHALER_NODE_VERSION"), os.Getenv("WHALER_BIN"))

	cmdArgs := os.Args[1:]

	if os.Getenv("WHALER_BIN") == "whaler" {
		if runtime.GOOS == "windows" && len(cmdArgs) == 0 {
			cmdArgs = append(cmdArgs, "-h")
		}
	}

	args = append(args, cmdArgs...)

	cmd, err := docker.Container(args)
	if err != nil {
		return err
	}

	if containerName != "" {
		killAppHandle(containerName)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()

	return err
}

func killAppHandle(name string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP)
	go func() {
		<-c
		for {
			docker.ContainerKill(name)
		}
	}()
}
