package main

import "os"
import "fmt"
import "net"
import "errors"
import "regexp"
import "os/exec"
import "runtime"
import "syscall"
import "strconv"
import "strings"
import "github.com/fatih/color"
import "github.com/fatih/flags"
import "golang.org/x/crypto/ssh/terminal"

func main() {
    var err interface{Error() string} = nil

    err = prepareAppEnv()

    if err == nil {
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

        checkErr := appContainerExists()
        if checkErr != nil {
            err = createAppContainer()
            if err == nil {
                if version == "" {
                    version = "latest"
                }
                err = setupApp("whaler_install", version)
            }

        } else if version != "" {
            err = setupApp("whaler_update", version)
        }

        if err == nil {
            err = runApp()
        }
    }

    if err != nil {
        if len(err.Error()) > 0 {
            red := color.New(color.FgRed).SprintFunc()
            fmt.Fprintf(os.Stderr, red("%v\n"), err)
        }
        os.Exit(1)
    }
}

func createCommand(name string, args []string) (*exec.Cmd, error) {
    path, lookErr := exec.LookPath(name)
    if lookErr != nil {
        return nil, lookErr
    }
    cmd := exec.Command(path, args...)
    cmd.Stderr = os.Stderr

    return cmd, nil
}

func docker(args []string) (*exec.Cmd, error) {
    cmd, err := createCommand("docker", args)
    if err != nil {
        return nil, err
    }
    cmd.Env = os.Environ()

    return cmd, nil
}

func dockerMachine(args []string) (*exec.Cmd, error) {
    return createCommand("docker-machine", args)
}

func convertWindowsToUnixPath(path string) string {
    path = strings.Replace(path, ":\\", "/", -1)
    path = strings.Replace(path, "\\", "/", -1)
    return "/" + path
}

func prepareAppEnv() error {
    if runtime.GOOS == "windows" {
        PWD, _ := os.Getwd()
        os.Setenv("PWD", convertWindowsToUnixPath(PWD))
        os.Setenv("HOME", convertWindowsToUnixPath(os.Getenv("USERPROFILE")))
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
            if err == nil {
                tcpAddr := &net.TCPAddr {
                    IP: addrs[0].(*net.IPNet).IP,
                }
                os.Setenv("DOCKER_IP", strings.Split(tcpAddr.String(), ":")[0])
            }
        }
    }

    return nil
}

func appContainerExists() error {
    args := []string{"inspect", "--format", "{{ .Id }}", "whaler"}
    cmd, err := docker(args)
    if err == nil {
        cmd.Stderr = nil
        err = cmd.Run()
    }

    return err
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

func createAppContainer() error {
    if os.Getenv("DOCKER_MACHINE_NAME") != "" {
        err := prepareDockerMachine(os.Getenv("DOCKER_MACHINE_NAME"))
        if err != nil {
            return err
        }
    }

    args := []string{"create", "--name", "whaler",
    "-v", "/usr/local/bin",
    "-v", "/usr/local/lib/node_modules",
    "-v", "/etc/whaler:/etc/whaler",
    "-v", "/var/lib/whaler:/var/lib/whaler",
    "-v", "/var/lib/docker:/var/lib/docker",
    "-v", "/var/run/docker.sock:/var/run/docker.sock"}

    if os.Getenv("DOCKER_MACHINE_NAME") != "" {
        args = append(args, "-v", "/mnt/sda1:/mnt/sda1")
    }

    args = append(args, "node:4.2")

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stdout = os.Stdout
    err = cmd.Run()

    return err
}

func setupApp(name string, version string) error {
    args := []string{"run", "--name", name, "-t", "--rm",
    "--volumes-from", "whaler"}

    if version == "dev" {
        args = append(args, "-e", "WHALER_SETUP=dev", "node:4.2")
        args = append(args, "npm", "install", "-g", "https://github.com/whaler/whaler.git")
    } else {
        args = append(args, "node:4.2")
        args = append(args, "npm", "install", "-g", "whaler@" + version)
    }

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stdout = os.Stdout
    err = cmd.Run()

    return err
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

func runApp() error {
    args := []string{"run",
    "--pid", "host",
    "--volumes-from", "whaler",
    "-v", os.Getenv("HOME") + ":" + os.Getenv("HOME"),
    "-v", os.Getenv("HOME") + "/.whaler" + ":" + "/root/.whaler"}

    if os.Getenv("WHALER_PATH") != "" {
        args = append(args, "-v", os.Getenv("WHALER_PATH") + ":" + "/usr/local/lib/node_modules/whaler")
    }

    if os.Getenv("DOCKER_IP") != "" {
        args = append(args, "-e", "WHALER_DOCKER_IP=" + os.Getenv("DOCKER_IP"))
    }

    daemon := ""
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
    }

    if daemon != "" {
        args = append(args, "-e", "WHALER_DAEMON_APPS=" + os.Getenv("HOME") + "/apps")
        args = append(args, "-v", os.Getenv("HOME") + "/apps" + ":" + "/root/apps")

        args = append(args, "-w", "/root/apps")
        args = append(args, "--name", "whaler_daemon_" + strconv.Itoa(syscall.Getpid()))

        args = append(args, "-d", "--restart", "always")
        args = append(args, "-p", daemon + ":" + daemon)

    } else {
        args = append(args, "-w", os.Getenv("PWD"))
        args = append(args, "--name", "whaler_" + strconv.Itoa(syscall.Getpid()))

        frontend := getAppFrontend()
        args = append(args, "-e", "WHALER_FRONTEND=" + frontend)

        if frontend == "interactive" {
            args = append(args, "-i")
        }
        args = append(args, "-t", "--rm")
    }

    args = append(args, "node:4.2", "whaler")

    cmdArgs := os.Args[1:]
    if runtime.GOOS == "windows" && len(cmdArgs) == 0 {
        cmdArgs = append(cmdArgs, "-h")
    }

    args = append(args, cmdArgs...)

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    err = cmd.Run()

    return err
}
