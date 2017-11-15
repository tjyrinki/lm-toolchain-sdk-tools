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
 */
package main

import (
	"fmt"
	"os"
	"os/exec"

	"launchpad.net/gnuflag"
	"link-motion.com/lm-toolchain-sdk-tools"
)

const (
	ERR_NO_ACCESS    = 255
	ERR_NEEDS_FIXING = 254
	ERR_NO_BRIDGE    = 253
	ERR_NO_LXC       = 252
	ERR_NO_SETUP     = 251
	//ERR_UNKNOWN      = 200
)

type initializedCmd struct {
	ignoreBridgeCheck bool
}

func (c *initializedCmd) usage() string {
	return `Checks if the container backend is setup correctly.

lmsdk-target initialized`
}

func (c *initializedCmd) flags() {
	gnuflag.BoolVar(&c.ignoreBridgeCheck, "b", false, "Do not check for lxc bridge")
}

func (c *initializedCmd) run(args []string) error {

	//check for one of the lxc binaries
	fmt.Println("Checking for lxc...")
	if _, err := exec.LookPath("lxc-create"); err != nil {
		fmt.Fprintf(os.Stderr, "LXC does not seem to be installed, make sure the lxc tools are in PATH.\n")
		os.Exit(ERR_NO_LXC)
	}

	if !c.ignoreBridgeCheck {
		fmt.Println("Checking for lxc network bridge...")
		err := lm_sdk_tools.LxcBridgeConfigured()
		if err != nil {
			fmt.Printf("The network check failed with error: %v\n", err)
			os.Exit(ERR_NO_BRIDGE)
		}
		fmt.Println("LXC bridge is configured with a subnet.")
	} else {
		fmt.Println("Skipping bridge check.")
	}

	fmt.Println("Checking for subUID setup...")
	if _, _, err := lm_sdk_tools.GetOrCreateUidRange(false); err != nil {
		fmt.Printf("subUID setup check failed with error: %v\n", err)
		os.Exit(ERR_NO_SETUP)
	}

	fmt.Println("Checking for subGID setup...")
	if _, _, err := lm_sdk_tools.GetOrCreateGuidRange(false); err != nil {
		fmt.Printf("subGID setup check failed with error: %v\n", err)
		os.Exit(ERR_NO_SETUP)
	}

	fmt.Println("Checking for lxc usernet...")
	if err := lm_sdk_tools.HasLxcUsernet(); err != nil {
		fmt.Printf("lxc usernet check failed with error: %v\n", err)
		os.Exit(ERR_NO_SETUP)
	}

	fmt.Println("Checking for required directories...")
	if err := lm_sdk_tools.EnsureRequiredDirectoriesExist(false); err != nil {
		fmt.Printf("directory check failed with error: %v\n", err)
		os.Exit(ERR_NO_SETUP)
	}

	for _, fixable := range fixable_set {
		fixableErr := fixable.Check()
		if fixableErr != nil {
			fmt.Printf("Error: %v\n", fixableErr)
			os.Exit(ERR_NEEDS_FIXING)
		}
	}

	fmt.Println("Container backend is ready.")
	return nil
}
