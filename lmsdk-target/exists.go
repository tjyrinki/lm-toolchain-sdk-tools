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

	"gopkg.in/lxc/go-lxc.v2"
	"link-motion.com/lm-toolchain-sdk-tools"
)

type existsCmd struct {
}

func (c *existsCmd) usage() string {
	return `Checks if a container exists.

lmsdk-target exists container`
}

func (c *existsCmd) flags() {
}

func (c *existsCmd) run(args []string) error {
	if len(args) < 1 {
		PrintUsage(c)
		os.Exit(1)
	}

	container, err := lxc.NewContainer(args[0], lm_sdk_tools.LMTargetPath())
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	if !container.Defined() {
		return fmt.Errorf("Container not found")
	}

	println("Container exists")
	return nil
}
