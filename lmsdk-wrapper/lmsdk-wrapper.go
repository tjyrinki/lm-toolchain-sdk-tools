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

// #include "lmsdk-wrapper.h"
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/pborman/uuid"
	"gopkg.in/lxc/go-lxc.v2"
	"link-motion.com/lm-sdk-tools"
)

var container string
var containerRootfs string

var paths = []string{"var", "bin", "boot", "dev", "etc", "lib", "lib64", "media", "mnt", "opt", "proc", "root", "run", "sbin", "srv", "sys", "usr"}
var re = regexp.MustCompile("(^|[^\\w+]|\\s+|-\\w)\\/(" + strings.Join(paths, "|") + ")")

func mapAndWrite(line *bytes.Buffer, out io.WriteCloser) {
	in := string(line.Bytes())
	in = re.ReplaceAllString(in, "$1"+containerRootfs+"/$2")
	out.Write([]byte(in))
}

func mapFunc(in *os.File, output io.WriteCloser, wg *sync.WaitGroup) {
	readBuf := make([]byte, 1)
	var lineBuf bytes.Buffer
	defer in.Close()
	defer wg.Done()
	for {
		n, err := in.Read(readBuf)

		if err != nil {
			break
		}

		if n > 0 {
			lineBuf.Write(readBuf)
			if readBuf[0] == byte('\n') {
				mapAndWrite(&lineBuf, output)
				lineBuf.Truncate(0)
			}
		}
	}

	if lineBuf.Len() > 0 {
		mapAndWrite(&lineBuf, output)
	}
}

func executeCommand() int {
	//figure out the container we should execute the command in
	//the parent directories name is supposed to be named like it
	//toolpath := filepath.Base(os.Args[0])
	container = filepath.Base(filepath.Dir(os.Args[0]))
	var err error

	containerRootfs, err = lm_sdk_tools.ContainerRootfs(container)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not request container rootfs: %v\n", err)
		return 1
	}

	c, err := lm_sdk_tools.LoadLMContainer(container)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the Container: %v\n", err)
		return 1
	}

	err = lm_sdk_tools.BootContainerSync(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not start the Container: %v\n", err)
		return 1
	}

	cmdName := filepath.Base(os.Args[0])
	cmdArgs := os.Args[1:]

	if cmdName == "cmake" {
		killCache := true
		for _, opt := range cmdArgs {
			if opt == "--help" {
				killCache = false
				break
			} else if opt == "--build" {
				killCache = false
				break
			}
		}

		if killCache {
			cwd, _ := os.Getwd()
			if _, err := os.Stat(path.Join(cwd, "CMakeCache.txt")); err == nil {
				fmt.Printf("-- Removing build artifacts\n")
				_ = os.RemoveAll(path.Join(cwd, "CMakeFiles"))
				_ = os.Remove(path.Join(cwd, "CMakeCache.txt"))
				_ = os.Remove(path.Join(cwd, "cmake_install.cmake"))
				_ = os.Remove(path.Join(cwd, "Makefile"))
			}
		}
	}

	//build the command, sourcing the dotfiles to get a decent shell
	args := []string{}
	args = append(args, cmdName)
	args = append(args, cmdArgs...)

	//until LXD supports sending signals to processes we need to have a pidfile
	u1 := uuid.NewUUID()
	pidfile := fmt.Sprintf("/tmp/%x.pid", u1)
	program := ""

	/*
	   rcFiles := []string{"/etc/profile", "$HOME/.profile"}
	   cwd, _ := os.Getwd()

	   for _, rcfile := range rcFiles {
	       program += "test -f " + rcfile + " && . " + rcfile + "; "
	   }

	   //make sure the working directory is the same
	   program += "cd \"" + cwd + "\" && "
	*/

	//write the current shells PID into the pidfile
	program += fmt.Sprintf("echo $$ > %s; ", pidfile)

	//force C locale as QtCreator needs it
	program += " LC_ALL=C exec"

	for _, arg := range args {
		program += " " + lm_sdk_tools.QuoteString(arg)
	}

	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

		for {
			sig := <-ch

			c.Container.RunCommandStatus([]string{
				"/bin/bash",
				"-c",
				fmt.Sprintf("kill -%d -$(ps -o pgid= `cat %s` | grep -o '[0-9]*')", sig, pidfile),
			}, lxc.DefaultAttachOptions)
		}
	}()

	stdout_r, stdout_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating the stdout pipe: %v\n", err)
		return 1
	}

	stderr_r, stderr_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating the stderr pipe: %v\n", err)
		return 1
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go mapFunc(stdout_r, os.Stdout, &wg)
	go mapFunc(stderr_r, os.Stderr, &wg)

	cid, cgid, _, err := lm_sdk_tools.DistroToUserIds(c.Distribution)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return 1
	}

	options := lxc.DefaultAttachOptions
	options.ClearEnv = true
	options.UID = int(cid)
	options.GID = int(cgid)
	options.Cwd, _ = os.Getwd()
	options.StdinFd = os.Stdin.Fd()
	options.StderrFd = stderr_w.Fd()
	options.StdoutFd = stdout_w.Fd()

	exitCode, cerr := c.Container.RunCommandStatus(
		[]string{"/bin/bash", "-c", program},
		options)

	stdout_w.Close()
	stderr_w.Close()

	//wait for mapFunc to finish
	wg.Wait()

	//since the pidfile is created in /tmp and /tmp is mounted into the container
	//we can just delete the local file
	defer os.Remove(pidfile)

	if cerr != nil {
		return 1
	}
	return exitCode
}

func main() {
	xit := executeCommand()

	if xit < 0 {
		xit = 1
	} else {
		xit = int(C.get_WEXITSTATUS(C.int(xit)))
	}
	os.Exit(xit)
}
