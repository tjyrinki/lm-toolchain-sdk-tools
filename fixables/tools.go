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
package fixables

import (
	"fmt"
	"os"

	"path"

	"gopkg.in/lxc/go-lxc.v2"
	"link-motion.com/lm-sdk-tools"
)

type ToolsFixable struct {
	requiredTools []string
}

func NewToolsFixable() *ToolsFixable {
	return &ToolsFixable{
		requiredTools: []string{
			"qmake",
			"gcc",
			"g++",
			"make",
			"cmake",
			"rpmbuild",
		},
	}
}

func (this *ToolsFixable) run(container *lxc.Container, doFix bool) error {

	containerDir := path.Dir(container.ConfigFileName())

	wrapperPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Could not resolve the absolute pathname of the tool")
	}

	wrapperTool := path.Dir(wrapperPath) + "/lmsdk-wrapper"

	for _, tool := range this.requiredTools {
		toolPath := containerDir + "/" + tool

		targetPath, err := os.Readlink(toolPath)
		if err != nil || targetPath != wrapperTool {
			//a broken link will also report that it does not exist
			fmt.Printf("... Creating tool %s -> %s\n", tool, wrapperTool)
			os.Remove(toolPath)
			err = os.Symlink(wrapperTool, toolPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create tool: %s", toolPath)
			}
		}
	}
	return nil
}

func (c *ToolsFixable) CheckContainer(container string) error {
	cont, err := lxc.NewContainer(container, lm_sdk_tools.LMTargetPath())
	if err != nil {
		return err
	}

	if !cont.Defined() {
		return fmt.Errorf("Container %s not found.", container)
	}

	return c.run(cont, false)
}

func (c *ToolsFixable) FixContainer(container string) error {
	cont, err := lxc.NewContainer(container, lm_sdk_tools.LMTargetPath())
	if err != nil {
		return err
	}

	if !cont.Defined() {
		return fmt.Errorf("Container %s not found.", container)
	}

	return c.run(cont, true)
}

func (c *ToolsFixable) Check() error {
	fmt.Printf("Checking for missing tools...\n")

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

func (c *ToolsFixable) Fix() error {
	fmt.Printf("Fixing missing tools...\n")

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

func (*ToolsFixable) NeedsRoot() bool {
	return false
}
