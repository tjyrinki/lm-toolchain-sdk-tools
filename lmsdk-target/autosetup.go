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
 *
 * Based on cloud-init (lp:cloud-init):
 * Author: Stéphane Graber
 */
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"launchpad.net/gnuflag"

	"gopkg.in/lxc/go-lxc.v2"
	"link-motion.com/lm-sdk-tools"
)

type autosetupCmd struct {
	yes          bool
	ignoreBridge bool
}

func (c *autosetupCmd) usage() string {
	return `Creates a default config for the container backend.

lmsdk-target autosetup [-y] [-b]`
}

func (c *autosetupCmd) flags() {
	gnuflag.BoolVar(&c.yes, "y", false, "Assume yes to all questions.")
	gnuflag.BoolVar(&c.ignoreBridge, "b", false, "Do not setup lxc bridge")
}

func (c *autosetupCmd) run(args []string) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("This command needs to run as root")
	}
	if !c.yes {
		if !lm_sdk_tools.GetUserConfirmation("WARNING: This will override existing configurations and restart all your containers, are your sure?") {
			return fmt.Errorf("Cancelled by user.")
		}
	}

	lxcUser, err := lm_sdk_tools.LxcContainerUser()
	if err != nil {
		return err
	}

	containers := lxc.Containers(lm_sdk_tools.LMTargetPath())

	stoppedContainers := []*lxc.Container{}
	//first let stop the containers
	fmt.Println("Stopping containers:")
	for _, container := range containers {
		if container.State() != lxc.STOPPED {
			fmt.Printf("Stopping %s .....", container.Name())
			err := container.Stop()
			if err != nil {
				return fmt.Errorf("Could not stop container %s. error: %v.", container.Name, err)
			}
			stoppedContainers = append(stoppedContainers, &container)
			fmt.Print(" DONE\n")
		}
	}
	fmt.Println("All containers stopped.")

	fmt.Printf("\nCreating default network bridge .....\n")

	err = lm_sdk_tools.LxcBridgeConfigured()
	if err != nil && !c.ignoreBridge {
		subnet, err := c.detectSubnet()
		if err != nil {
			return err
		}

		err = c.editLXCBridgeFile(subnet)
		if err != nil {
			return err
		}

		fmt.Println("\nRestarting services:")
		cmd := exec.Command("bash", "-c", "systemctl enable lxc-net && systemctl restart lxc-net")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println(" FAILED")
			return fmt.Errorf("Restarting the LXC network service failed. error: %v", err)
		}

		fmt.Println(" DONE")
	} else {
		fmt.Println(" SKIPPED")
	}

	fmt.Printf("\nGenerating default ID mappings .....\n")
	if _, _, err := lm_sdk_tools.GetOrCreateUidRange(true); err != nil {
		fmt.Println(" FAILED")
		return fmt.Errorf("subUID setup failed with error: %v", err)
	}

	if _, _, err := lm_sdk_tools.GetOrCreateGuidRange(true); err != nil {
		fmt.Println(" FAILED")
		return fmt.Errorf("subGID setup failed with error: %v", err)
	}
	fmt.Println(" DONE")

	fmt.Printf("\nGenerating lxc-usernet settings .....\n")
	if err := lm_sdk_tools.EditLxcUsernet(); err != nil {
		fmt.Println(" FAILED")
		return fmt.Errorf("lxc-usernet setup failed with error: %v", err)
	}
	fmt.Println(" DONE")

	fmt.Printf("Setting up directories .....\n")
	if err := lm_sdk_tools.EnsureRequiredDirectoriesExist(true); err != nil {
		fmt.Println(" FAILED")
		return fmt.Errorf("Directory setup failed with error: %v", err)
	}
	fmt.Println(" DONE")

	if len(stoppedContainers) > 0 {
		fmt.Println("\nStarting previously stopped containers:")
		for _, container := range stoppedContainers {
			fmt.Printf("Starting %s .....", container.Name())

			cmd := exec.Command("sudo", "-u", lxcUser.Username, "--",
				"lxc-start", "-P", lm_sdk_tools.LMTargetPath(), "-n", container.Name())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				fmt.Printf(" FAILED\n%v\n", err)
			} else {
				fmt.Print(" DONE\n")
			}
		}

	}
	return nil
}

func (c *autosetupCmd) detectSubnet() (string, error) {
	used := make([]int, 0)

	ipAddrOutput, err := exec.Command("ip", "addr", "show").CombinedOutput()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(ipAddrOutput), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		columns := strings.Split(trimmed, " ")

		if len(columns) < 2 {
			return "", fmt.Errorf("invalid ip addr output line %s", line)
		}

		if columns[0] != "inet" {
			continue
		}

		addr := columns[1]
		if !strings.HasPrefix(addr, "10.0.") {
			continue
		}

		tuples := strings.Split(addr, ".")
		if len(tuples) < 4 {
			return "", fmt.Errorf("invalid ip addr %s", addr)
		}

		subnet, err := strconv.Atoi(tuples[2])
		if err != nil {
			return "", err
		}

		used = append(used, subnet)
	}

	curr := 1
	for {
		isUsed := false
		for _, subnet := range used {
			if subnet == curr {
				isUsed = true
				break
			}
		}
		if !isUsed {
			break
		}

		curr++
		if curr > 254 {
			return "", fmt.Errorf("No valid subnet available")
		}
	}

	return fmt.Sprintf("%d", curr), nil
}

func (c *autosetupCmd) editLXCBridgeFile(subnet string) error {
	buffer := bytes.Buffer{}

	f, err := os.OpenFile(lm_sdk_tools.LxcBridgeFile, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	input := string(data)
	newValues := map[string]string{
		"USE_LXC_BRIDGE": "true",
		"LXC_BRIDGE":     "lxcbr0",
		"LXC_ADDR":       fmt.Sprintf("10.0.%s.1", subnet),
		"LXC_NETMASK":    "255.255.255.0",
		"LXC_NETWORK":    fmt.Sprintf("10.0.%s.1/24", subnet),
		"LXC_DHCP_RANGE": fmt.Sprintf("10.0.%s.2,10.0.%s.254", subnet, subnet),
		"LXC_DHCP_MAX":   "253",
	}

	found := map[string]bool{}

	for _, line := range strings.Split(input, "\n") {
		out := line

		if !strings.HasPrefix(line, "#") {
			for prefix, value := range newValues {
				if strings.HasPrefix(line, prefix+"=") {
					out = fmt.Sprintf(`%s="%s"`, prefix, value)
					found[prefix] = true
					break
				}
			}
		}

		buffer.WriteString(out)
		buffer.WriteString("\n")
	}

	for prefix, value := range newValues {
		if !found[prefix] {
			buffer.WriteString(prefix)
			buffer.WriteString("=")
			buffer.WriteString(value)
			buffer.WriteString("\n")
			found[prefix] = true // not necessary but keeps "found" logically consistent
		}
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = f.WriteString(buffer.String())
	if err != nil {
		return err
	}
	return nil
}
