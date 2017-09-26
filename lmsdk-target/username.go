/*
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

	"launchpad.net/gnuflag"

	"link-motion.com/lm-sdk-tools"
)

type usernameCmd struct {
	container string
}

func (c *usernameCmd) usage() string {
	return `Returns the default user name inside the container.
 
 lmsdk-target username container`
}

func (c *usernameCmd) flags() {
}

func (c *usernameCmd) run(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, c.usage())
		gnuflag.PrintDefaults()
		return fmt.Errorf("Missing arguments.")
	}

	c.container = args[0]

	container, err := lm_sdk_tools.LoadLMContainer(c.container)
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	if !container.Container.Defined() {
		return fmt.Errorf("Container does not exist")
	}

	_, _, username, err := lm_sdk_tools.DistroToUserIds(container.Distribution)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", username)
	return nil
}
