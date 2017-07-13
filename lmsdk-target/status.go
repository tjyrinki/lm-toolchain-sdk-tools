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
	"log"
	"os"
	"time"

	"launchpad.net/gnuflag"

	"gopkg.in/lxc/go-lxc.v2"
	"link-motion.com/lm-sdk-tools"

	"encoding/json"
)

type statusCmd struct {
	container string
}

func (c *statusCmd) usage() string {
	return `Shows the current status of the container.

lmsdk-target status container`
}

func (c *statusCmd) flags() {
}

func (c *statusCmd) run(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, c.usage())
		gnuflag.PrintDefaults()
		return fmt.Errorf("Missing arguments.")
	}

	c.container = args[0]

	container, err := lxc.NewContainer(c.container, lm_sdk_tools.LMTargetPath())
	if err != nil {
		return fmt.Errorf("ERROR: %s", err.Error())
	}

	if !container.Defined() {
		return fmt.Errorf("Container does not exist")
	}

	info := container.State()

	if container.State() != lxc.RUNNING {
		return fmt.Errorf("Container is not running")
	}

	result := make(map[string]string)
	result["status"] = info.String()

	if _, err := container.WaitIPAddresses(5 * time.Second); err != nil {
		log.Fatalf("Could not query IP addresses: %v\n", err)
	}

	ips, err := container.IPv4Address("eth0")
	if err != nil {
		log.Fatalf("Could not query IP addresses: %v\n", err)
	}
	result["ipv4"] = ips[0]

	js, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("Could not marshal the result into a valid json string. error: %v.", err)
	}
	fmt.Printf("%s\n", js)
	return nil
}
