/*
 * Copyright (C) 2017 Link Motion Oy.
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
	"strings"

	"launchpad.net/gnuflag"
	"link-motion.com/lm-toolchain-sdk-tools"
)

type rpmInstall struct {
	container         string
	preferredpackages string
	noninteractive    bool
}

func (c *rpmInstall) usage() string {
	return `Installs rpm packages into the container.
	
	lmsdk-target rpminstall <container> <package1> [package2 package3 ...] `
}

func (c *rpmInstall) flags() {
	gnuflag.StringVar(&c.preferredpackages, "preferredpackages", "", "Directory of packages to be preferred during build.")
	gnuflag.BoolVar(&c.noninteractive, "non-interactive", false, "Do not ask anything, use default answers automatically.")
}

func (c *rpmInstall) run(args []string) error {
	//need at least the container name and one package to install
	if len(args) < 2 {
		PrintUsage(c)
		return fmt.Errorf("Container name missing")
	}
	c.container = args[0]

	//make sure the container exists
	container, err := lm_sdk_tools.LoadLMContainer(c.container)
	if err != nil {
		return fmt.Errorf("Could not connect to the Container: %v", err)
	}

	if len(c.preferredpackages) > 0 {
		err, repoDir := lm_sdk_tools.AddZypperRepository(c.preferredpackages, "rpminstallpackages", 20, false, container)
		if err != nil {
			fmt.Printf("Adding repository failed: %v\n", err)
			return err
		}

		defer func() {
			lm_sdk_tools.RemoveZypperRepository("rpminstallpackages", container)
			os.RemoveAll(repoDir)
		}()
	}

	commandStr := "zypper install %s"
	if c.noninteractive {
		commandStr = "zypper --non-interactive install %s"
	}

	exitCode, err := lm_sdk_tools.RunInContainer(
		container,
		true,
		[]string{},
		fmt.Sprintf(commandStr, strings.Join(args[1:], " ")),
		os.Stdout.Fd(),
		os.Stderr.Fd(),
	)
	if err != nil || exitCode != 0 {
		return fmt.Errorf("Failed to install packages.\n")
	}
	return nil
}
