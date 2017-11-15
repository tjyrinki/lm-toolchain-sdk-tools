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
	"time"

	lxc "gopkg.in/lxc/go-lxc.v2"

	"launchpad.net/gnuflag"
	"link-motion.com/lm-toolchain-sdk-tools"
)

type snapshotCmd struct {
	container    string
	snapshotName string
	restore      bool
	reset        bool
	destroy      bool
	list         bool
}

func (c *snapshotCmd) usage() string {
	return `Creates a snapshot of the container.
 
 lmsdk-target snapshot <container>`
}

func (c *snapshotCmd) flags() {
	gnuflag.StringVar(&c.snapshotName, "N", "", "snapshot name")
	gnuflag.BoolVar(&c.restore, "restore", false, "Restore last snapshot, or the one specified by -N")
	gnuflag.BoolVar(&c.reset, "reset", false, "Reset container to first taken snapshot, discarding all others")
	gnuflag.BoolVar(&c.destroy, "destroy", false, "Destroy last snapshot, or the one given by -N")
	gnuflag.BoolVar(&c.list, "list", false, "List all snapshots")
}

func (c *snapshotCmd) run(args []string) error {

	if len(args) < 1 {
		PrintUsage(c)
		return fmt.Errorf("Container name missing")
	}
	c.container = args[0]

	//make sure the container exists
	container, err := lm_sdk_tools.LoadLMContainer(c.container)
	if err != nil {
		return fmt.Errorf("Could not connect to the Container: %v", err)
	}

	createSnap := !c.restore && !c.reset && !c.destroy && !c.list
	startContainer := false

	if createSnap || c.reset || c.restore || c.destroy {
		if container.Container.State() != lxc.STOPPED {
			fmt.Printf("Stopping container...\n")
			startContainer = true
			container.Container.Stop()
		}
	}

	defer func() {
		if startContainer {
			fmt.Printf("Starting container...\n")
			if err := lm_sdk_tools.BootContainerSync(container); err != nil {
				fmt.Printf("Failed to restart container: %v\n", err)
			}
		}
	}()

	if c.restore || c.reset {
		if c.destroy || c.list || (c.restore && c.reset) {
			PrintUsage(c)
			os.Exit(1)
		}

		snaps, err := container.Container.Snapshots()
		if err != nil {
			return fmt.Errorf("Failed to load existing snapshots: %v", err)
		}

		var snapToRes *lxc.Snapshot
		if c.snapshotName != "" {
			for i, snap := range snaps {
				if c.snapshotName == snap.Name {
					snapToRes = &snaps[i]
					break
				}
			}
		} else {
			searchMode := newestSnapshot
			if c.reset {
				searchMode = oldestSnapshot
			}

			idx, err := findSnapshot(&snaps, searchMode)
			if err != nil {
				return err
			}
			snapToRes = &snaps[idx]
		}

		if snapToRes == nil {
			fmt.Printf("No snapshot available\n")
			return nil
		}

		fmt.Printf("Restoring Snap %s with Time %v\n", snapToRes.Name, snapToRes.Timestamp)
		err = container.Container.RestoreSnapshot(*snapToRes, container.Name)
		if err != nil {
			return fmt.Errorf("Failed to restore snapshot: %v", err)
		}

		//after a snapshot was loaded all LM specific files are gone, recreate them
		err = FinalizeContainer(container)
		if err != nil {
			return fmt.Errorf("Failed to write LM specific config: %v", err)
		}

		//delete all other snapshots if reset
		if c.reset {
			fmt.Printf("Removing all other snapshots\n")
			for _, snap := range snaps {
				if snap.Name == snapToRes.Name {
					continue
				}
				err = container.Container.DestroySnapshot(snap)
				if err != nil {
					//failing to delete a snapshot is not critical, just continue
					fmt.Fprintf(os.Stderr, "Failed to delete snapshot: %s: %v\n", snap.Name, err)
				}
			}
		}
	} else if c.list {
		if c.destroy {
			PrintUsage(c)
			os.Exit(1)
		}

		snaps, err := container.Container.Snapshots()
		if err != nil {
			return fmt.Errorf("Failed to load existing snapshots: %v", err)
		}

		for _, snap := range snaps {
			fmt.Printf("%s %s\n", snap.Name, snap.Timestamp)
		}
	} else if c.destroy {
		snaps, err := container.Container.Snapshots()
		if err != nil {
			return fmt.Errorf("Failed to load existing snapshots: %v", err)
		}

		var snapToDel *lxc.Snapshot
		if c.snapshotName != "" {
			for i, snap := range snaps {
				if c.snapshotName == snap.Name {
					snapToDel = &snaps[i]
					break
				}
			}
		} else {
			idx, err := findSnapshot(&snaps, newestSnapshot)
			if err != nil {
				return err
			}
			snapToDel = &snaps[idx]
		}
		fmt.Printf("Deleting Snap %s with Time %v\n", snapToDel.Name, snapToDel.Timestamp)
		err = container.Container.DestroySnapshot(*snapToDel)
		if err != nil {
			return fmt.Errorf("Failed to delete snapshot: %v", err)
		}
	} else {
		fmt.Printf("Creating snapshot...\n")
		snap, err := container.Container.CreateSnapshot()
		if err != nil {
			return fmt.Errorf("Failed to create the snapshot: %v", err)
		}

		fmt.Printf("Created snapshot: %s\n", snap.Name)
		return nil
	}
	return nil
}

type findMode int

const (
	// find oldest snapshot
	oldestSnapshot findMode = iota + 1
	// find newest snapshot
	newestSnapshot
)

func findSnapshot(snapshots *[]lxc.Snapshot, mode findMode) (int, error) {
	currSnap := -1
	var currTime time.Time
	var err error

	for i, snap := range *snapshots {
		if currSnap < 0 {
			currSnap = i
			currTime, err = time.Parse("2006:01:02 15:04:05", snap.Timestamp)
			if err != nil {
				return -1, fmt.Errorf("Failed to parse timestamp %s: %v", (*snapshots)[currSnap].Timestamp, err)
			}
			continue
		}

		datetime, err := time.Parse("2006:01:02 15:04:05", snap.Timestamp)
		if err != nil {
			return -1, fmt.Errorf("Failed to parse timestamp %s: %v", snap.Timestamp, err)
		}

		switch mode {
		case oldestSnapshot:
			if datetime.Before(currTime) {
				currSnap = i
				currTime = datetime
			}
		case newestSnapshot:
			if datetime.After(currTime) {
				currSnap = i
				currTime = datetime
			}
		}

	}
	return currSnap, nil
}
