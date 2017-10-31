package main

import "io"
import "os"
import "fmt"
import "net"
import "time"
import "bufio"
import "bytes"
import "errors"
import "regexp"
import "os/exec"
import "runtime"
import "syscall"
import "strconv"
import "strings"
import "net/http"
import "io/ioutil"
import "crypto/md5"
import "encoding/hex"
import "github.com/fatih/color"
import "github.com/nareix/curl"
import "github.com/fatih/flags"
import "github.com/Jeffail/gabs"
import "github.com/kardianos/osext"
import "golang.org/x/crypto/ssh/terminal"
import "github.com/inconshreveable/go-update"

const NODE_VERSION = "6.9.1"

const DOWNLOAD_URL = "https://github.com/whaler/whaler-client/releases/download/"

// Return cursor to start of line and clean it
const RESET_LINE = "\r\033[K\r"

func main() {
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

        checkErr := appContainerExists()
        if checkErr != nil {
            if version == "" {
                version = "latest"
            }
            doSetup = true

        } else if version != "" {
            doSetup = true
        }

        if doSetup {
            updated, permissionsErr := trySelfUpdate()

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
                if nodeVersion, errVersion := getContainerNodeVersion(); errVersion == nil {
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

func sudoSelfUpdate() {
    if selfPath, pathErr := getSelfPath(); pathErr == nil {
        args := []string{"-E", selfPath, "self-update"}
        cmd, cmdErr := createCommand("sudo", args)

        if cmdErr != nil {
            printErrorAndExit(cmdErr)
        }

        cmd.Env = os.Environ()
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        err := cmd.Run()

        if err != nil {
            printErrorAndExit(err)
        }
    }
}

func makeSelfUpdate() (bool, error) {
    updated := false

    updateOpts := update.Options{}

    if permissionsErr := updateOpts.CheckPermissions(); permissionsErr != nil {
        return false, permissionsErr
    }

    fmt.Printf("Please wait...")
    file := "whaler"
    if runtime.GOOS == "windows" {
        file = "whaler.exe"
    }
    err := doUpdate(DOWNLOAD_URL + runtime.GOOS + "_" + runtime.GOARCH + "/" + file)
    fmt.Printf(RESET_LINE)

    if err == nil {
        updated = true
    }

    return updated, nil
}

func trySelfUpdate() (bool, error) {
    var permissionsErr interface{Error() string} = nil

    updated := false

    if selfPath, pathErr := getSelfPath(); pathErr == nil {
        if md5sum, md5Err := computeMd5(selfPath); md5Err == nil {

            fmt.Printf("Please wait...")
            remoteMd5sum, _ := remoteMd5(DOWNLOAD_URL + runtime.GOOS + "_" + runtime.GOARCH + "/md5")
            fmt.Printf(RESET_LINE)

            if "" != remoteMd5sum && md5sum != remoteMd5sum {
                c := askForConfirmation("New version of `whaler-client` available. Download it?", "y")
                if c {
                    updated, permissionsErr = makeSelfUpdate()
                    if permissionsErr != nil {
                        return false, permissionsErr
                    }
                }
            }
        }
    }

    return updated, nil
}

func getSelfPath() (string, error) {
    filename, err := osext.Executable()
    return filename, err
}

func computeMd5(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    var result []byte

    hash := md5.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }

    return hex.EncodeToString(hash.Sum(result)), nil
}

func remoteMd5(url string) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil  {
        return "", err
    }

    s := strings.Split(string(body[:]), " = ")

    if len(s) == 2 {
        return strings.TrimSpace(s[1]), nil
    }

    return "", nil
}

func askForConfirmation(msg string, onEmpty string) bool {
    reader := bufio.NewReader(os.Stdin)

    if onEmpty != "n" {
        onEmpty = "y"
    }

    for {
        if onEmpty == "n" {
            fmt.Printf("%s [y/N]: ", msg)
        } else {
            fmt.Printf("%s [Y/n]: ", msg)
        }

        response, _ := reader.ReadString('\n')
        response = strings.ToLower(strings.TrimSpace(response))
        fmt.Printf("\033[1A") // Move up 1 line
        fmt.Printf(RESET_LINE)

        if response == "" {
            response = onEmpty
        }

        if response == "y" || response == "yes" {
            return true
        } else if response == "n" || response == "no" {
            return false
        }
    }
}

func doUpdate(url string) error {
    req := curl.New(url)

    msgLen := 0
    req.Progress(func (p curl.ProgressStatus) {
        if p.Size > 0 {
            fmt.Printf(RESET_LINE)
            msg := fmt.Sprintf("Downloading...[%s/%s]", curl.PrettySizeString(p.Size), curl.PrettySizeString(p.ContentLength))
            if runtime.GOOS == "windows" {
                fmt.Printf("%s\r", strings.Repeat(" ", msgLen))
                msgLen = len(msg)
            }
            fmt.Printf(msg)
        }
    }, time.Second)

    resp, err := req.Do()
    if err != nil {
        return err
    }

    if 200 == resp.StatusCode {
        body := bytes.NewReader([]byte(resp.Body))

        err = update.Apply(body, update.Options{})
    } else {
        err = errors.New(strconv.Itoa(resp.StatusCode))
    }

    return err
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

func getContainerNodeVersion() (string, error) {
    args := []string{"run", "-t", "--rm",
    "--volumes-from", "whaler", 
    "node:" + os.Getenv("WHALER_NODE_VERSION"),
    "node", "-v"}
    cmd, err := docker(args)
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

func removeAppContainer() error {
    args := []string{"rm", "-f", "whaler"}

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    err = cmd.Run()

    return err
}

func createWhalerDir(path string) error {
    args := []string{"run", "--rm",
    "-v", path + ":/.whaler_tmp",
    "node:" + os.Getenv("WHALER_NODE_VERSION"),
    "mkdir", "-p", "/.whaler_tmp/whaler"}

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    //cmd.Stdout = os.Stdout
    err = cmd.Run()

    return err
}

func createWhalerNetwork() error {
    args := []string{"network", "create", "whaler_nw"}

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stderr = nil
    err = cmd.Run()

    return err
}

func inspectWhalerNetwork() error {
    args := []string{"network", "inspect", "whaler_nw"}

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stderr = nil
    err = cmd.Run()

    return err
}

func createAppContainer() error {
    if os.Getenv("DOCKER_MACHINE_NAME") != "" {
        err := prepareDockerMachine(os.Getenv("DOCKER_MACHINE_NAME"))
        if err != nil {
            return err
        }
    }

    etcWhaler := "/etc/whaler:/etc/whaler";
    if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
        etcWhaler = "/etc/whaler";
    } else {
        createWhalerDir("/etc");
    }

    createWhalerDir("/var/lib");

    args := []string{"create", "--name", "whaler",
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

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stdout = os.Stdout
    err = cmd.Run()

    return err
}

func setupApp(version string) error {
    args := []string{"run", "--name", "whaler_setup", "-t", "--rm",
    "--volumes-from", "whaler"}

    if version == "dev" {
        args = append(args, "-e", "WHALER_SETUP=dev", "node:" + os.Getenv("WHALER_NODE_VERSION"))
        args = append(args, "npm", "install", "https://github.com/whaler/whaler.git")
    } else {
        args = append(args, "node:" + os.Getenv("WHALER_NODE_VERSION"))
        args = append(args, "npm", "install", "whaler@" + version)
    }
    args = append(args, "--global", "--production", "--unsafe-perm", "--loglevel=error", "--no-update-notifier")

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
    "-e", "HOME=" + os.Getenv("WHALER_HOME"),
    "-v", os.Getenv("HOME") + ":" + os.Getenv("HOME"),
    "-v", os.Getenv("HOME") + "/.whaler" + ":" + os.Getenv("WHALER_HOME") + "/.whaler"}

    content, readErr := ioutil.ReadFile(os.Getenv("HOME") + "/.whaler/client.json")
    if readErr == nil {
        parsed, parseErr := gabs.ParseJSON(content)
        if parseErr != nil {
            return parseErr
        }

        children, _ := parsed.S("file-sharing").Children()
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
        args = append(args, "-w", os.Getenv("PWD"))
        args = append(args, "--name", "whaler_" + strconv.Itoa(syscall.Getpid()))

        frontend := getAppFrontend()
        args = append(args, "-e", "WHALER_FRONTEND=" + frontend)

        if frontend == "interactive" {
            args = append(args, "-it")
        }
        args = append(args, "--rm")
    }

    useWhalerNetwork := false
    if err := inspectWhalerNetwork(); err == nil {
        useWhalerNetwork = true
    } else if err := createWhalerNetwork(); err == nil {
        useWhalerNetwork = true
    }
    if useWhalerNetwork {
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

    cmd, err := docker(args)
    if err != nil {
        return err
    }
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    err = cmd.Run()

    return err
}
