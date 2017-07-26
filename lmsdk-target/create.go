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
 * Based on the usdk-target code
 */

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"

	"path"

	"encoding/json"

	"io/ioutil"

	"gopkg.in/lxc/go-lxc.v2"
	"launchpad.net/gnuflag"
	"link-motion.com/lm-sdk-tools"
	"link-motion.com/lm-sdk-tools/fixables"
)

type createCmd struct {
	buildArchitecture string
	hostArchitecture  string
	distro            string
	version           string
	name              string
	createSupGroups   bool
	enableUpdates     bool
}

func (c *createCmd) usage() string {
	return `Creates a new Link Motion SDK build target.

lmsdk-target create -n NAME -d DISTRO -v VERSION -a ARCH
`
}

var requiredString = "REQUIRED"
var baseFWRegexNoMinor = regexp.MustCompile("^(ubuntu-[^-]+-[\\d]{1,2}\\.[\\d]{1,2})-([^-]+)-([^-]+)-([^-]+)?$")
var baseFWRegexWithMinor = regexp.MustCompile("^(ubuntu-[^-]+-[\\d]{1,2}\\.[\\d]{1,2})(\\.[\\d]+)-([^-]+)-([^-]+)-([^-]+)?$")

func (c *createCmd) flags() {
	gnuflag.StringVar(&c.buildArchitecture, "b", requiredString, "Build architecture of the target")
	gnuflag.StringVar(&c.hostArchitecture, "a", requiredString, "Host architecture of the image")
	gnuflag.StringVar(&c.distro, "d", requiredString, "Distribution (ivios or autoos)")
	gnuflag.StringVar(&c.version, "v", requiredString, "Image release version")
	gnuflag.StringVar(&c.name, "n", requiredString, "name of the container")
	gnuflag.BoolVar(&c.createSupGroups, "g", false, "Also try to create the users supplementary groups")
}

func (c *createCmd) run(args []string) error {
	if c.hostArchitecture == requiredString || c.distro == requiredString ||
		c.version == requiredString || c.name == requiredString ||
		c.buildArchitecture == requiredString {
		gnuflag.PrintDefaults()
		return fmt.Errorf("Missing arguments")
	}

	if os.Getuid() != 0 {
		//return fmt.Errorf("This command needs to run as root")
	}

	container, err := lxc.NewContainer(c.name, lm_sdk_tools.LMTargetPath())
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	mapfile, err := c.GenerateDefaultConfigFile(c.distro)
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	err = container.LoadConfigFile(mapfile)
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	container.SetVerbosity(lxc.Verbose)

	template := "/opt/lm-sdk/bin/lxc-lm-download"
	execname, err := os.Executable()
	templateAlt := path.Join(path.Dir(execname), "lxc-lm-download")

	if _, err := os.Stat(templateAlt); err == nil {
		template = templateAlt
	} else if _, err := os.Stat(template); os.IsNotExist(err) {
		return fmt.Errorf("The lxc-lm-download was not found on the system")
	}

	downloader := path.Join(path.Dir(template), "lmsdk-download")
	if _, err := os.Stat(downloader); os.IsNotExist(err) {
		return fmt.Errorf("The lmsdk-download tool was not found on the system")
	}

	options := lxc.TemplateOptions{
		Template:             template,
		Distro:               c.distro,
		Release:              c.version,
		Arch:                 c.hostArchitecture,
		Variant:              c.buildArchitecture,
		FlushCache:           true,
		DisableGPGValidation: true,
	}

	options.ExtraArgs = append(options.ExtraArgs, fmt.Sprintf("--downloader=%s", downloader))

	containerUserId, _, containerUserName, err := lm_sdk_tools.DistroToUserIds(c.distro)
	if err != nil {
		return err
	}

	if err := container.Create(options); err != nil {
		lm_sdk_tools.RemoveContainerSync(container.Name())
		return fmt.Errorf("ERROR: %v", err.Error())
	}

	if err = c.registerUserInContainer(container, uint(containerUserId), containerUserName); err != nil {
		lm_sdk_tools.RemoveContainerSync(container.Name())
		return fmt.Errorf("ERROR: %v", err.Error())
	}

	tools := fixables.NewToolsFixable()
	if err = tools.FixContainer(c.name); err != nil {
		lm_sdk_tools.RemoveContainerSync(container.Name())
		return fmt.Errorf("ERROR: %v", err.Error())
	}

	//everything worked out, as last write the config-lm file
	lmContainer := lm_sdk_tools.LMTargetContainer{
		Name:           c.name,
		Architecture:   c.buildArchitecture,
		Version:        c.version,
		Distribution:   c.distro,
		UpdatesEnabled: false,
		Container:      nil,
	}

	lmConfig, err := json.MarshalIndent(&lmContainer, "  ", "  ")
	if err != nil {
		lm_sdk_tools.RemoveContainerSync(container.Name())
		return fmt.Errorf("Unable to marshall config-lm file: %v", err.Error())
	}

	err = ioutil.WriteFile(container.ConfigFileName()+"-lm", lmConfig, 0664)
	if err != nil {
		lm_sdk_tools.RemoveContainerSync(container.Name())
		return fmt.Errorf("Unable to write config-lm file: %v", err.Error())
	}

	return nil
}

func (c *createCmd) GenerateDefaultConfigFile(distro string) (string, error) {
	confDir, err := lm_sdk_tools.ConfigPath()
	if err != nil {
		return "", err
	}

	confFileName := fmt.Sprintf("%s/lmsdk-%s-default.conf", confDir, distro)
	if _, err := os.Stat(confFileName); os.IsNotExist(err) {

		fmt.Printf("Creating %s\n", confFileName)

		confFile, err := os.Create(confFileName)
		if err != nil {
			return "", err
		}

		defer confFile.Close()
		writer := bufio.NewWriter(confFile)

		currUser, err := user.Current()
		if err != nil {
			return "", err
		}

		t_uid, err := strconv.ParseUint(currUser.Uid, 10, 32)
		if err != nil {
			return "", err
		}

		t_gid, err := strconv.ParseUint(currUser.Gid, 10, 32)
		if err != nil {
			return "", err
		}

		containerUid, containerGid, _, err := lm_sdk_tools.DistroToUserIds(c.distro)
		if err != nil {
			return "", err
		}

		uid := uint32(t_uid)
		gid := uint32(t_gid)

		firstUid, uidRange, err := lm_sdk_tools.GetOrCreateUidRange(false)
		if err != nil {
			return "", err
		}

		firstGid, gidRange, err := lm_sdk_tools.GetOrCreateGuidRange(false)
		if err != nil {
			return "", err
		}

		writer.WriteString("lxc.include = /etc/lxc/default.conf\n")

		if containerUid >= firstUid && containerUid < (firstUid+uidRange) {

		}

		//map the first uid and gid range before the current users id
		writer.WriteString(fmt.Sprintf("lxc.id_map = u 0 %d %d\n", firstUid, containerUid))
		writer.WriteString(fmt.Sprintf("lxc.id_map = g 0 %d %d\n", firstGid, containerGid))

		//now the user ID is mapped 1:1
		writer.WriteString(fmt.Sprintf("lxc.id_map = u %d %d 1\n", containerUid, uid))
		writer.WriteString(fmt.Sprintf("lxc.id_map = g %d %d 1\n", containerGid, gid))

		//and the rest
		writer.WriteString(fmt.Sprintf("lxc.id_map = u %d %d %d\n", containerUid+1, firstUid+containerUid+1, uidRange-containerUid-1))
		writer.WriteString(fmt.Sprintf("lxc.id_map = g %d %d %d\n", containerGid+1, firstGid+containerGid+1, gidRange-containerGid-1))
		writer.Flush()
	}

	return confFileName, nil
}

func (c *createCmd) registerUserInContainer(container *lxc.Container, containerUserId uint, containerUserName string) error {

	currUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("cannot get current user: %v", err)
	}

	pw, err := lm_sdk_tools.Getpwnam(currUser.Username)
	if err != nil {
		return fmt.Errorf("Querying the user entry failed. error: %v", err)
	}

	if pw.Uid == 0 {
		return fmt.Errorf("Registering root is not possible")
	}

	groups, err := lm_sdk_tools.GetGroups()
	if err != nil {
		return fmt.Errorf("Querying the group entry failed. error: %v", err)
	}

	var requiredGroups []lm_sdk_tools.GroupEntry
	for _, group := range groups {
		if group.Gid == pw.Gid {
			requiredGroups = append(requiredGroups, group)
			break
		}
	}

	if container.State() != lxc.STOPPED {
		err = container.Stop()
		if err != nil {
			return err
		}
	}

	//add the home dir
	homedir := pw.Dir
	err = container.SetConfigItem("lxc.mount.entry", fmt.Sprintf("%s %s none bind,create=dir 0 0", homedir, homedir[1:]))
	if err != nil {
		return err
	}

	err = container.SetConfigItem("lxc.mount.entry", "/tmp tmp none bind,create=dir 0 0")
	if err != nil {
		return err
	}

	err = container.SetConfigItem("lxc.mount.entry", "/media media none bind,create=dir 0 0")
	if err != nil {
		return err
	}

	err = container.SaveConfigFile(container.ConfigFileName())
	if err != nil {
		return err
	}

	err = container.Start()
	if err != nil {
		return err
	}
	/*
		fmt.Printf("Creating groups\n")
		for _, group := range requiredGroups {
			mustWork := group.Gid == pw.Gid

			fmt.Printf("Creating group %s\n", group.Name)

			cmd := exec.Command("lxc-attach", "-P", lm_sdk_tools.LMTargetPath(), "-n", container.Name(),
				"--", "groupadd", "-g", strconv.FormatUint(uint64(group.Gid), 10), group.Name)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err := cmd.Wait(); err != nil {
				print("GroupAdd returned error\n")
				if exiterr, ok := err.(*exec.ExitError); ok {
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						//exit code of 9 means the group exists already
						//which we will treat as success
						if status.ExitStatus() != 9 {
							if mustWork {
								return fmt.Errorf("Could not create primary group")
							}
							continue
						}
					}
				} else {
					return fmt.Errorf("Failed to add the group %s. error: %v", group.Name, err)
				}
			}
		}

		fmt.Printf("Creating user %s\n", pw.LoginName)

		command := []string{
			"-P", lm_sdk_tools.LMTargetPath(), "-n", container.Name(), "--",
			"useradd", "--no-create-home",
			"-u", strconv.FormatUint(uint64(pw.Uid), 10),
			"--gid", strconv.FormatUint(uint64(pw.Gid), 10),
			"--home-dir", pw.Dir,
			"-s", "/bin/bash",
			"-p", "*",
			"--groups", "video",
			pw.LoginName,
		}

		cmd := exec.Command("lxc-attach", command...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

	*/

	command := []string{
		"-P", lm_sdk_tools.LMTargetPath(), "-n", container.Name(), "--",
		"sed", "-i",
		fmt.Sprintf("s;/home/%s;/home/%s;", containerUserName, pw.LoginName),
		"/etc/passwd",
	}

	cmd := exec.Command("lxc-attach", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
