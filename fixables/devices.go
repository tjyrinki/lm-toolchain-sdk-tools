/*
 * Copyright (C) 2016 Canonical Ltd
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
 * Author: Benjamin Zeller <benjamin.zeller@canonical.com>
 */
package fixables

import (
	"fmt"
	"os"

	"bufio"
	"bytes"
	"io/ioutil"
	"strings"

	"gopkg.in/lxc/go-lxc.v2"
	"link-motion.com/lm-sdk-tools"
)

type DevicesFixable struct{}

func (c *DevicesFixable) run(container *lxc.Container, doFix bool) error {

	//first check the mounts
	var cleanDevices []string
	var brokenDevices []string
	errorFound := false

	for _, mountpt := range container.ConfigItem("lxc.mount.entry") {
		mount := strings.Fields(mountpt)

		if len(mount) != 6 {
			fmt.Printf("Invalid mountpoint found: %s\n", mountpt)
			continue
		}

		if _, err := os.Stat(mount[0]); os.IsNotExist(err) && mount[2] != "tmpfs" {
			errorFound = true
			brokenDevices = append(brokenDevices, mount[0])
		} else {
			cleanDevices = append(cleanDevices, mountpt)
		}
	}

	if errorFound {
		if doFix {
			fmt.Printf("Devices %s does not exist on the host.\n", brokenDevices)

			//read config file
			conf, err := ioutil.ReadFile(container.ConfigFileName())
			if err != nil {
				return fmt.Errorf("Unable to read container config: %s", err)
			}

			fi, err := os.Create(container.ConfigFileName())
			if err != nil {
				return fmt.Errorf("Unable to write container config: %s", err)
			}

			writer := bufio.NewWriter(fi)

			reader := bufio.NewScanner(bytes.NewReader(conf))
			for reader.Scan() {
				line := reader.Text()
				for _, mountpt := range brokenDevices {
					if !strings.Contains(line, mountpt) {
						writer.WriteString(fmt.Sprintf("%s\n", line))
					}
				}
			}

			writer.Flush()
			fi.Close()

			container.ClearConfig()
			container.LoadConfigFile(container.ConfigFileName())

		} else {
			return fmt.Errorf("Devices %s does not exist on the host.", brokenDevices)
		}
	}
	return nil
}

func (c *DevicesFixable) CheckContainer(container string) error {
	cont, err := lxc.NewContainer(container, lm_sdk_tools.LMTargetPath())
	if err != nil {
		return err
	}

	if !cont.Defined() {
		return fmt.Errorf("Container %s not found.", container)
	}

	return c.run(cont, false)
}

func (c *DevicesFixable) FixContainer(container string) error {
	cont, err := lxc.NewContainer(container, lm_sdk_tools.LMTargetPath())
	if err != nil {
		return err
	}

	if !cont.Defined() {
		return fmt.Errorf("Container %s not found.", container)
	}

	return c.run(cont, true)
}

func (c *DevicesFixable) Check() error {
	fmt.Printf("Checking for broken devices...\n")

	targets, err := lm_sdk_tools.FindLMTargets()
	if err != nil {
		return err
	}

	for _, target := range targets {
		err = c.run(target.Container, false)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *DevicesFixable) Fix() error {
	fmt.Println("Checking for and removing broken devices....")
	targets, err := lm_sdk_tools.FindLMTargets()
	if err != nil {
		return err
	}

	for _, target := range targets {
		err = c.run(target.Container, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (*DevicesFixable) NeedsRoot() bool {
	return false
}
