/*
 * Copyright (C) 2016 Canonical Ltd
 * Copyright (C) 2016 Link Motion Oy
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
 */
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"launchpad.net/gnuflag"
	"link-motion.com/lm-toolchain-sdk-tools"
)

type execCmd struct {
	maintMode bool
	container string
	user      string
}

func (c *execCmd) usage() string {
	myMode := "exec"
	if c.maintMode {
		myMode = "maint"
	}

	return fmt.Sprintf(`Executes a command in the container.

lmsdk-target %s <container> [command]`, myMode)
}

func (c *execCmd) flags() {
	gnuflag.StringVar(&c.user, "u", "", "Username to login before executing the command.")
}

func (c *execCmd) run(args []string) error {
	if len(args) < 1 {
		PrintUsage(c)
		os.Exit(1)
	}

	c.container = args[0]
	args = args[1:]

	if len(c.user) == 0 {
		lmCont, err := lm_sdk_tools.LoadLMContainer(c.container)
		if err != nil {
			return err
		}
		_, _, cUser, err := lm_sdk_tools.DistroToUserIds(lmCont.Distribution)
		if err != nil {
			return err
		}

		c.user = cUser
	}

	lxc_command, err := exec.LookPath("lxc-attach")
	if err != nil {
		return err
	}

	lxc_args := []string{
		lxc_command, "-P", lm_sdk_tools.LMTargetPath(),
		"-n", c.container, "--",
		"su",
	}

	if len(args) == 0 {
		lxc_args = append(lxc_args, "-l")
	}

	lxc_args = append(lxc_args, []string{
		"-s", "/bin/bash"}...)

	if !c.maintMode {
		lxc_args = append(lxc_args, c.user)
	}

	if len(args) > 0 {
		rcFiles := []string{"/etc/profile", "$HOME/.profile"}
		cwd, _ := os.Getwd()

		program := ""
		for _, rcfile := range rcFiles {
			program += "test -f " + rcfile + " && . " + rcfile + " &> /dev/null; "
		}

		//make sure the working directory is the same
		program += "cd \"" + cwd + "\" && "

		//force C locale as QtCreator needs it
		program += " LC_ALL=C "

		for _, arg := range args {
			program += " " + lm_sdk_tools.QuoteString(arg)
		}

		lxc_args = append(lxc_args, []string{
			"-c", program}...)
	}

	os.Stdout.Sync()
	os.Stderr.Sync()
	err = syscall.Exec(lxc_command, lxc_args, os.Environ())
	fmt.Printf("Error: %v\n", err)
	return nil
}
