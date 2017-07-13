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
 * Based on the LXD lxc client. Copyright holders:
 * Author: Chris Glass
 * Author: Gustavo Niemeyer
 * Author: Jeroen Simonetti
 * Author: Matt Morrison
 * Author: René Jochum
 * Author: Serge Hallyn
 * Author: Stéphane Graber
 * Author: Tycho Andersen
 */
package main

import (
	"fmt"
	"os"
	"strings"

	"launchpad.net/gnuflag"
	"link-motion.com/lm-sdk-tools"
)

var errArgs = fmt.Errorf("wrong number of subcommand arguments")

type command interface {
	usage() string
	flags()
	run(args []string) error
}

var commands = map[string]command{
	"list":        &listCmd{},
	"help":        &helpCmd{},
	"create":      &createCmd{},
	"rootfs":      &rootfsCmd{},
	"status":      &statusCmd{},
	"exists":      &existsCmd{},
	"maint":       &execCmd{maintMode: true},
	"exec":        &execCmd{maintMode: false},
	"run":         &execCmd{maintMode: false},
	"destroy":     &destroyCmd{},
	"images":      &imagesCmd{},
	"upgrade":     &upgradeCmd{},
	"initialized": &initializedCmd{},
	"autosetup":   &autosetupCmd{},
	"autofix":     &autofixCmd{},
	//"set" : &setCmd{},
}

func main() {
	if err := run(); err != nil {
		// The action we take depends on the error we get.
		msg := fmt.Sprintf("error: %v", err)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("%s", msg))
		os.Exit(1)
	}
}

func run() error {
	var err error

	lm_sdk_tools.EnsureLXCInitializedOrDie()

	if len(os.Args) < 2 {
		commands["help"].run(nil)
		os.Exit(1)
	}

	//origArgs := os.Args
	name := os.Args[1]
	cmd, ok := commands[name]
	if !ok {
		commands["help"].run(nil)
		fmt.Fprintf(os.Stderr, "\nerror: unknown command: %s\n", name)
		os.Exit(1)
	}

	cmd.flags()
	gnuflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n\nOptions:\n\n", strings.TrimSpace(cmd.usage()))
		gnuflag.PrintDefaults()
	}

	os.Args = os.Args[1:]
	gnuflag.Parse(true)

	err = cmd.run(gnuflag.Args())
	if err == errArgs {
		fmt.Fprintf(os.Stderr, "%s\n\nerror: %v\n", cmd.usage(), err)
		os.Exit(1)
	}
	return err
}
