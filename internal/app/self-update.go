package app

import (
	"io"
	"os"
	"fmt"
	"time"
	"bytes"
	"bufio"
	"errors"
	"runtime"
	"strconv"
	"strings"
	"net/http"
	"io/ioutil"
	"archive/tar"
	"compress/gzip"
	_ "unsafe" // for go:linkname

	"github.com/nareix/curl"
	"github.com/kardianos/osext"
	"github.com/inconshreveable/go-update"
)

//go:linkname GOARM runtime.goarm
var GOARM uint8

const DOWNLOAD_BUCKET = "https://github.com/whaler/whaler-client/releases/download/0.x"

func generateDownloadUrl() string {
	url := fmt.Sprintf("%s/whaler_%s_%s", DOWNLOAD_BUCKET, runtime.GOOS, runtime.GOARCH)

	if runtime.GOARCH == "arm" {
		url = fmt.Sprintf("%s%d", url, GOARM)
	}

	return fmt.Sprintf("%s.tar.gz", url)
}

func doUpdate() error {
	req := curl.New(generateDownloadUrl())

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

		gzipReader, err := gzip.NewReader(body)
		if err != nil {
			return err
		}
		defer gzipReader.Close()

		tarReader := tar.NewReader(gzipReader)

		filename := "whaler"
		if runtime.GOOS == "windows" {
			filename += ".exe"
		}

		var fileReader io.Reader = nil

		for true {
			header, tarErr := tarReader.Next()

			if tarErr == io.EOF {
				break
			}

			if tarErr != nil {
				return tarErr
			}

			if filename == header.Name && header.Typeflag == tar.TypeReg {
				fileReader = tarReader
				break
			}
		}

		if fileReader == nil {
			return fmt.Errorf("Binary file not found")
		}

		err = update.Apply(fileReader, update.Options{})

	} else {
		err = errors.New(strconv.Itoa(resp.StatusCode))
	}

	return err
}

func makeSelfUpdate() (bool, error) {
	updated := false

	updateOpts := update.Options{}

	if permissionsErr := updateOpts.CheckPermissions(); permissionsErr != nil {
		return false, permissionsErr
	}

	fmt.Printf("Please wait...")
	err := doUpdate()
	fmt.Printf(RESET_LINE)

	if err == nil {
		updated = true
	}

	return updated, nil
}

func getSelfPath() (string, error) {
	filename, err := osext.Executable()
	return filename, err
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

func getRemoteVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/version", DOWNLOAD_BUCKET))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil  {
		return "", err
	}

	return strings.TrimSpace(string(body[:])), nil
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

func trySelfUpdate(appVersion string) (bool, error) {
	var permissionsErr interface{Error() string} = nil

	updated := false

	fmt.Printf("Please wait...")
	remoteVersion, _ := getRemoteVersion()
	fmt.Printf(RESET_LINE)

	if "" != remoteVersion && appVersion != remoteVersion {
		c := askForConfirmation("New version of `whaler-client` available. Download it?", "y")
		if c {
			updated, permissionsErr = makeSelfUpdate()
			if permissionsErr != nil {
				return false, permissionsErr
			}
		}
	}

	return updated, nil
}
