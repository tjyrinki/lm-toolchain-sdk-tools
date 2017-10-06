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
	"os"

	"link-motion.com/lm-sdk-tools"
)

type destroyCmd struct {
	container string
}

func (c *destroyCmd) usage() string {
	return `Deletes a container.

lmsdk-target destroy container`
}

func (c *destroyCmd) flags() {
}

func (c *destroyCmd) run(args []string) error {
	if len(args) < 1 {
		PrintUsage(c)
		os.Exit(1)
	}
	c.container = args[0]

	return lm_sdk_tools.RemoveContainerSync(c.container)
}
