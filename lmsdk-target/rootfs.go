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

	"link-motion.com/lm-toolchain-sdk-tools"
)

type rootfsCmd struct {
}

func (c *rootfsCmd) usage() string {
	return `Shows the path to the root filesystem of a container.

lmsdk-target rootfs container`
}

func (c *rootfsCmd) flags() {
}

func (c *rootfsCmd) run(args []string) error {
	if len(args) < 1 {
		PrintUsage(c)
		os.Exit(1)
	}

	rootfs, err := lm_sdk_tools.ContainerRootfs(args[0])
	if err != nil {
		return err
	}

	fmt.Printf(rootfs + "\n")
	return nil
}
