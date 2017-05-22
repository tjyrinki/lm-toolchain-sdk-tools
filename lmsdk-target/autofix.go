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

import "link-motion.com/lm-sdk-tools/fixables"

var fixable_set = []fixables.Fixable{
	&fixables.DevicesFixable{},
	fixables.NewToolsFixable(),
	/*
	   &fixables.ContainerAccess{},
	   &fixables.DRIFixable{},
	   &fixables.NvidiaFixable{},
	*/
}

type autofixCmd struct {
}

func (c *autofixCmd) usage() string {
	return `Automatically fixes problems in the container backends.`
}

func (c *autofixCmd) flags() {
}

func (c *autofixCmd) run(args []string) error {

	for _, fixable := range fixable_set {
		err := fixable.Fix()
		if err != nil {
			return err
		}
	}

	/*
	   targets, err := ubuntu_sdk_tools.FindClickTargets()
	   if err != nil {
	       return err
	   }

	   for _, target := range targets {
	       err = ubuntu_sdk_tools.UpdateConfigSync(client, target.Name)
	       if err != nil {
	           return err
	       }
	   }
	*/
	return nil
}
