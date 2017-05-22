/*
 * Copyright (C) 2016 Canonical Ltd
 * Copyright (C) 2017 Link-Motion Oy.
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
 * Based on the LXD lxc client. Copyright holders:
 * Author: Gustavo Niemeyer
 * Author: S.Çağlar Onur
 * Author: Stéphane Graber
 * Author: Tycho Andersen
 */
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
)

type helpCmd struct {
}

func (c *helpCmd) usage() string {
	return "Presents details on how to use lmsdk-target."
}

func (c *helpCmd) flags() {
}

func (c *helpCmd) run(args []string) error {
	if len(args) > 0 {
		for _, name := range args {
			cmd, ok := commands[name]
			if !ok {
				fmt.Fprintf(os.Stderr, "error: unknown command: %s\n", name)
			} else {
				fmt.Fprintf(os.Stderr, cmd.usage()+"\n")
			}
		}
		return nil
	}

	fmt.Println("Usage: lmsdk-target [subcommand] [options]")
	fmt.Println("Available commands:")
	var names []string
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := commands[name]
		fmt.Printf("\t%-10s - %s\n", name, c.summaryLine(cmd.usage()))
	}

	//fmt.Println("Environment:")
	//fmt.Println("  LXD_CONF           " + "Path to an alternate client configuration directory.")
	//fmt.Println("  LXD_DIR            " + "Path to an alternate server directory.")
	return nil
}

// summaryLine returns the first line of the help text. Conventionally, this
// should be a one-line command summary, potentially followed by a longer
// explanation.
func (c *helpCmd) summaryLine(usage string) string {
	usage = strings.TrimSpace(usage)
	s := bufio.NewScanner(bytes.NewBufferString(usage))
	if s.Scan() {
		if len(s.Text()) > 1 {
			return s.Text()
		}
	}
	return "Missing summary."
}
