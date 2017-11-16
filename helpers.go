/*
 * Copyright (C) 2016 Canonical Ltd
 * Copyright (C) 2017 Link Motion Oy
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * Author: Benjamin Zeller <benjamin.zeller@link-motion.com>
 *
 * Based on the ubuntu-sdk-tools
 */
package lm_sdk_tools

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"bufio"

	"strconv"

	"encoding/json"

	"gopkg.in/lxc/go-lxc.v2"
)

type LMTargetContainer struct {
	Name           string         `json:"name"`
	Architecture   string         `json:"architecture"`
	Distribution   string         `json:"distribution"`
	Version        string         `json:"version"`
	UpdatesEnabled bool           `json:"updatesEnabled"`
	Container      *lxc.Container `json:"-"`
}

const LxcBridgeFile = "/etc/default/lxc-net"
const LxcUsernetFile = "/etc/lxc/lxc-usernet"
const LmImageServerEnvVar = "LM_IMAGE_SERVER"

func LMTargetPath() string {
	user, err := LxcContainerUser()
	if err != nil {
		fmt.Printf("Fatal: Could not query the current user.")
		os.Exit(1)
	}

	return "/var/lib/lm-sdk/" + user.Username + "/containers"
}

func EnsureLXCInitializedOrDie() error {
	//set up /etc/lxc/lxc-usernet and subgid subuid
	return nil
}

func GetOrCreateUidRange(doCreate bool) (uint32, uint32, error) {
	currUser, err := LxcContainerUser()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get user: %v", err)
	}

	return GetOrCreateIdRange(currUser.Username, "/etc/subuid", doCreate)
}

func GetOrCreateGuidRange(doCreate bool) (uint32, uint32, error) {

	currUser, err := LxcContainerUser()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get user: %v", err)
	}
	return GetOrCreateIdRange(currUser.Username, "/etc/subgid", doCreate)
}

func GetOrCreateIdRange(user string, fileName string, doCreate bool) (uint32, uint32, error) {

	mode := os.O_RDONLY
	if doCreate {
		mode = os.O_RDWR | os.O_CREATE
	}

	file, err := os.OpenFile(fileName, mode, 0644)
	if err != nil {
		return 0, 0, err
	}

	defer file.Close()

	nextSubIdStart := uint32(100000)
	reader := bufio.NewScanner(file)
	for reader.Scan() {
		line := reader.Text()

		values := strings.Split(line, ":")
		if len(values) != 3 {
			continue
		}

		idStart, err := strconv.ParseUint(values[1], 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid number in %s %v", fileName, err)
		}

		idCount, err := strconv.ParseUint(values[2], 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid number in %s %v", fileName, err)
		}

		//if our user has already a entry, we do not need to care
		if values[0] == user {
			fmt.Printf("Found ID range: %d %d\n", idStart, idCount)
			return uint32(idStart), uint32(idCount), nil
		}

		nextId := uint32(idStart) + uint32(idCount)
		if nextId > nextSubIdStart {
			nextSubIdStart = nextId
		}
	}

	if os.Getuid() != 0 || !doCreate {
		return 0, 0, fmt.Errorf("No sub id was found, please run lmsdk-target initialize.")
	}

	fmt.Printf("Create ID range: %d 65536\n", nextSubIdStart)
	fmt.Fprintf(file, "%s:%d:65536", user, nextSubIdStart)
	return nextSubIdStart, 65536, nil

}

func BootContainerSync(container *LMTargetContainer) error {
	switch container.Container.State() {
	case lxc.STARTING:
		container.Container.Wait(lxc.RUNNING, time.Second*5)
	case lxc.STOPPING:
		container.Container.Wait(lxc.STOPPED, time.Second*5)
	case lxc.FREEZING:
		container.Container.Wait(lxc.FROZEN, time.Second*5)
	case lxc.ABORTING:
		fallthrough
	case lxc.THAWED:
		return fmt.Errorf("Container in unsupported state")
	}

	if container.Container.State() != lxc.RUNNING {
		err := container.Container.Start()
		if err != nil {
			return fmt.Errorf("Error while starting the container: %v\n", err)
		}
		container.Container.Wait(lxc.RUNNING, time.Second*5)
	}
	return nil
}

func StopContainerSync(container string) error {
	return nil
}

func UpdateConfigSync(container string) error {
	return nil
}

func RemoveContainerSync(container string) error {
	c, err := lxc.NewContainer(container, LMTargetPath())
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	if c.State() != lxc.STOPPED {
		c.Stop()
	}

	if err := c.Destroy(); err != nil {
		return fmt.Errorf("Cold not remove the target: %s", err.Error())
	}
	return nil
}

func GetUserConfirmation(question string) bool {
	var response string
	responses := map[string]bool{
		"y": true, "yes": true,
		"n": false, "no": false,
	}

	ok := false
	answer := false
	for !ok {
		fmt.Print(question + " (yes/no): ")
		_, err := fmt.Scanln(&response)
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(response)
		answer, ok = responses[response]
	}

	return answer
}

func ContainerRootfs(container string) (string, error) {
	c, err := lxc.NewContainer(container, LMTargetPath())
	if err != nil {
		return "", fmt.Errorf("ERROR: %s", err.Error())
	}

	if !c.Defined() {
		return "", fmt.Errorf("Container %s does not exist", container)
	}

	return c.ConfigItem("lxc.rootfs")[0], nil
}

func toLmContainer(c *lxc.Container) (*LMTargetContainer, error) {
	if !c.Defined() {
		return nil, fmt.Errorf("Container %s does not exist", c.Name())
	}

	//read config file
	conf, err := ioutil.ReadFile(c.ConfigFileName() + "-lm")
	if err != nil {
		return nil, fmt.Errorf("Unable to read container config file: %s", err)
	}

	lmContainer := LMTargetContainer{
		Container: nil,
	}

	err = json.Unmarshal(conf, &lmContainer)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse container config file: %s", err)
	}

	lmContainer.Name = c.Name()
	lmContainer.Container = c
	return &lmContainer, nil
}

func LoadLMContainer(container string) (*LMTargetContainer, error) {
	c, err := lxc.NewContainer(container, LMTargetPath())
	if err != nil {
		return nil, fmt.Errorf("ERROR: %s", err.Error())
	}
	return toLmContainer(c)
}

func FindLMTargets() ([]LMTargetContainer, error) {

	all_containers := lxc.Containers(LMTargetPath())
	lmTargets := []LMTargetContainer{}

	for _, container := range all_containers {

		lmContainer, err := toLmContainer(container)
		if err != nil {
			return nil, err
		}

		lmTargets = append(lmTargets,
			*lmContainer,
		)
	}

	return lmTargets, nil
}

func ConfigPath() (string, error) {

	currUser, err := user.Current()
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/.config/lm-sdk", currUser.HomeDir)

	if err = os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	return path, nil
}

func LxcContainerUser() (*user.User, error) {
	if os.Getuid() == 0 {
		key := "SUDO_UID"
		env := os.Getenv(key)

		if len(env) == 0 {
			key = "PKEXEC_UID"
			env = os.Getenv(key)
			if len(env) == 0 {
				return nil, fmt.Errorf("Running from root login not supported, use sudo to execute.")
			}
		}

		user, err := user.LookupId(env)
		if err != nil {
			return nil, fmt.Errorf("Os environment var :%s contains a invalid USER ID. error: %v", key, err)
		}

		return user, nil
	}

	currUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("cannot get current user: %v", err)
	}

	return currUser, nil
}

func ReadLxcBridgeConfig() (map[string]string, error) {

	f, err := os.Open(LxcBridgeFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	requiredValues := map[string]string{
		"USE_LXC_BRIDGE": "",
		"LXC_BRIDGE":     "",
		"LXC_NETWORK":    "",
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}

		dataSet := strings.Split(line, "=")
		if len(dataSet) != 2 {
			continue
		}

		prefix := strings.TrimSpace(dataSet[0])
		data := strings.TrimSpace(dataSet[1])
		data = strings.Trim(data, "\"")

		_, ok := requiredValues[prefix]
		if ok {
			fmt.Printf("Key %v has value \"%v\".\n", prefix, data)
			requiredValues[prefix] = data
		}

	}
	return requiredValues, nil
}

func LxcBridgeConfigured() error {

	requiredValues, err := ReadLxcBridgeConfig()
	if err != nil {
		return err
	}

	if requiredValues["USE_LXC_BRIDGE"] != "true" ||
		requiredValues["LXC_BRIDGE"] == "" ||
		requiredValues["LXC_NETWORK"] == "" {
		return fmt.Errorf("lxc-bridge not configured")
	}

	return nil
}

func ReadLxcUsernetConfig() ([][]string, error) {
	f, err := os.Open(LxcUsernetFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var values [][]string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 4 || strings.HasPrefix(line, "#") {
			continue
		}
		values = append(values, fields)
	}

	return values, nil
}

func HasLxcUsernet() error {
	set, err := ReadLxcUsernetConfig()
	if err != nil {
		return err
	}

	user, err := LxcContainerUser()
	if err != nil {
		return err
	}

	for _, entry := range set {
		if entry[0] == user.Username {
			return nil
		}
	}

	return fmt.Errorf("User not found in usernet config")
}

func EditLxcUsernet() error {
	if err := HasLxcUsernet(); err == nil {
		return nil
	}

	user, err := LxcContainerUser()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(LxcUsernetFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = LxcBridgeConfigured(); err != nil {
		return fmt.Errorf("Setting up the LXC usernet requires a configured bridge: %v", err)
	}

	bridgeConf, err := ReadLxcBridgeConfig()
	if err != nil {
		return err
	}

	fmt.Fprintf(f, "%s veth %s 999\n", user.Username, bridgeConf["LXC_BRIDGE"])
	return nil
}

func EnsureRequiredDirectoriesExist(doFix bool) error {
	targetPath := LMTargetPath()
	userPath := filepath.Dir(targetPath)
	rootPath := filepath.Dir(userPath)

	user, err := LxcContainerUser()
	if err != nil {
		return fmt.Errorf("Querying the user failed: %v", err)
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return fmt.Errorf("Invalid User ID: %v", err)
	}
	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return fmt.Errorf("Invalid Group ID: %v", err)
	}

	if err = EnsureDirExistsWithPermissions(rootPath, 0, 0, os.ModeDir|0755, doFix); err != nil {
		return err
	}

	if err = EnsureDirExistsWithPermissions(userPath, uid, gid, os.ModeDir|0750, doFix); err != nil {
		return err
	}

	if err = EnsureDirExistsWithPermissions(targetPath, uid, gid, os.ModeDir|0750, doFix); err != nil {
		return err
	}
	return nil
}

func EnsureDirExistsWithPermissions(dirName string, ownerUid, ownerGid int, perm os.FileMode, doFix bool) error {

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if !doFix {
			return fmt.Errorf("Directory does not exist: %v", err)
		}
		if err = os.MkdirAll(dirName, perm); err != nil {
			return fmt.Errorf("Creating the required directory failed: %v", err)
		}
	}

	fileInfo, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		return fmt.Errorf("The directory does not exist: %v", err)
	}

	fileUid := fileInfo.Sys().(*syscall.Stat_t).Uid
	fileGid := fileInfo.Sys().(*syscall.Stat_t).Gid

	if ownerUid != int(fileUid) || ownerGid != int(fileGid) {
		if !doFix {
			return fmt.Errorf("Wrong owner for directory: %s", dirName)
		}
		if err = os.Chown(dirName, ownerUid, ownerGid); err != nil {
			return fmt.Errorf("Changing the directory ownership failed: %v", err)
		}
	}

	if fileInfo.Mode() != perm {
		if !doFix {
			return fmt.Errorf("Wrong permissions for directory: %s", dirName)
		}
		if err = os.Chmod(dirName, perm); err != nil {
			return fmt.Errorf("Changing the directory permissions failed: %v", err)
		}
	}

	return nil
}

func DistroToUserIds(distro string) (uint32, uint32, string, error) {
	return 20000, 1002, "system", nil
	/*
		if distro == "link-motion-autoos" {
			return 20000, 1002, "org.c4c.ui_cluster", nil
		} else if distro == "link-motion-ivios" {
			return 20000, 1002, "system", nil
		} else {
			return 0, 0, "", fmt.Errorf("Unknown distro: %s", distro)
		}
	*/
}

func RunInContainer(c *LMTargetContainer, runAsRoot bool, env []string, program string, stdoutFd uintptr, stderrFd uintptr) (int, error) {
	cid, cgid, _, err := DistroToUserIds(c.Distribution)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return 1, nil
	}

	options := lxc.DefaultAttachOptions
	options.ClearEnv = true
	if runAsRoot {
		options.UID = int(0)
		options.GID = int(0)
		env = append(env, "HOME=/root")
	} else {
		options.UID = int(cid)
		options.GID = int(cgid)
		currUser, err := user.Current()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query current user")
			return 0, err
		}
		env = append(env, fmt.Sprintf("HOME=%s", currUser.HomeDir))
	}

	options.Cwd, _ = os.Getwd()
	options.StdinFd = os.Stdin.Fd()

	if stdoutFd >= 0 {
		options.StdoutFd = stdoutFd
	} else {
		options.StdoutFd = os.Stdout.Fd()
	}
	if stderrFd >= 0 {
		options.StderrFd = stderrFd
	} else {
		options.StderrFd = os.Stderr.Fd()
	}

	fullcmd := ""
	//we need to set the env variables right in the command, otherwise the bash --login will override them
	if len(env) > 0 {
		envList := ""
		for _, envVar := range env {
			envList += "\"" + envVar + "\" "
		}
		fullcmd = fmt.Sprintf("env %s ", envList)
	}

	fullcmd = fullcmd + program

	fmt.Printf("Running command: %s\n", fullcmd)

	return c.Container.RunCommandStatus(
		[]string{"/bin/bash", "--login", "-c", fullcmd},
		options)

}

func RunInContainerOuput(c *LMTargetContainer, runAsRoot bool, env []string, program string) ([]string, int, error) {

	stdout_r, stdout_w, err := os.Pipe()
	if err != nil {
		return []string{}, 1, fmt.Errorf("Error creating the stdout output pipe: %v\n", err)
	}

	stderr_r, stderr_w, err := os.Pipe()
	if err != nil {
		return []string{}, 1, fmt.Errorf("Error creating the stderr output pipe: %v\n", err)
	}

	defer stdout_r.Close()
	defer stderr_r.Close()

	exitCode, err := RunInContainer(c, runAsRoot, env, program, stdout_w.Fd(), stderr_w.Fd())
	stdout_w.Close()
	stderr_w.Close()

	output := make([]string, 2)

	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout_r)
	output[0] = buf.String() // Does a complete copy of the bytes in the buffer.

	buf.Reset()
	buf.ReadFrom(stderr_r)
	output[1] = buf.String() // Does a complete copy of the bytes in the buffer.

	return output, exitCode, err
}

/*
AddZypperRepository Initializes a zypper repository, adds it to the container and updates it.

sourceDir = Plain directory with packages to copy to the repository. Not touched by this function.
name = Name of the repository (don't use spaces!)
priority = Zypper priority (lower is higher!)
runUpdate = run zypper update if true
container = Container to add the repository into

The repository is created as temporary directory in /tmp. Caller must delete it after use
if needed. Old repository with same name is removed before adding the new one.
Requires 'createrepo' command to be available.

Returns directory of the created repository and error code.
*/
func AddZypperRepository(sourceDir string, name string, priority int, runUpdate bool, container *LMTargetContainer) (error, string) {
	repoDir, err := ioutil.TempDir("", "lm-sdk-repo"+name)
	out, err := exec.Command("bash", "-c", "cp "+filepath.Join(sourceDir, "*")+" "+repoDir).CombinedOutput()
	if err != nil {
		fmt.Printf("Copying files to repository failed: %v, %s", err, out)
		return err, repoDir
	}
	_, err = RunInContainer(container, true, []string{}, "zypper --non-interactive rr "+name, os.Stdout.Fd(), os.Stderr.Fd())
	if err != nil {
		return fmt.Errorf("Failed to execute zypper rr command in the container: %v", err), repoDir
	}
	_, err = RunInContainer(container, true, []string{}, "zypper --non-interactive ar -p "+strconv.Itoa(priority)+" -G "+repoDir+" "+name, os.Stdout.Fd(), os.Stderr.Fd())
	if err != nil {
		return fmt.Errorf("Failed to execute zypper ar command in the container: %v", err), repoDir
	}
	if runUpdate {
		_, err = RunInContainer(container, true, []string{}, "zypper --non-interactive up", os.Stdout.Fd(), os.Stderr.Fd())
		if err != nil {
			return fmt.Errorf("Failed to execute zypper up command in the container: %v", err), repoDir
		}
	}
	return nil, repoDir
}

/*
RemoveZypperRepository Remove a zypper repository.

name = Name of the repository (don't use spaces!)
container = Container to remove the repository from
*/
func RemoveZypperRepository(name string, container *LMTargetContainer) error {
	_, err := RunInContainer(container, true, []string{}, "zypper --non-interactive rr "+name, os.Stdout.Fd(), os.Stderr.Fd())
	if err != nil {
		return fmt.Errorf("Failed to execute zypper rr command in the container: %v", err)
	}
	return nil
}
