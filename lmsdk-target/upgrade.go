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
)

type upgradeCmd struct {
}

func (c *upgradeCmd) usage() string {
	return `Upgrades the container.

lmsdk-target upgrade container`
}

func (c *upgradeCmd) flags() {
}

func (c *upgradeCmd) run(args []string) error {
	if len(args) < 1 {
		PrintUsage(c)
		os.Exit(1)
	}

	exec := &execCmd{maintMode: true}

	execArgs := []string{
		args[0],
		"/bin/bash", "-c", "zypper --non-interactive ref && zypper --non-interactive up -y",
	}

	return exec.run(execArgs)
}
