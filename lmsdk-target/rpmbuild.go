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
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"launchpad.net/gnuflag"
	"link-motion.com/lm-sdk-tools"
)

type rpmbuildCmd struct {
	container       string
	projectDir      string
	specfile        string
	tarballName     string
	outputDirectory string
	jobs            int
	installDeps     bool
	upgrade         bool
}

func (c *rpmbuildCmd) usage() string {
	return (`Build a rpm in a container build target.

lmsdk-target rpmbuild container <sourcedir> [-t tarballname] [-j threads] [-s specfile]`)
}

func (c *rpmbuildCmd) flags() {
	gnuflag.StringVar(&c.specfile, "s", "", "specfile location")
	gnuflag.StringVar(&c.tarballName, "t", "", "tarball name")
	gnuflag.IntVar(&c.jobs, "j", runtime.NumCPU(), "The number of threads to pass to make")
	gnuflag.StringVar(&c.outputDirectory, "o", "", "Output directory where rpm files are placed")
	gnuflag.BoolVar(&c.installDeps, "build-deps", false, "Install build dependencies")
	gnuflag.BoolVar(&c.upgrade, "upgrade-before", false, "Upgrade container before starting the build")
}

/**
 * Queries information from the specfile, pass query in rpm QUERYFORMAT
 */
func (c *rpmbuildCmd) rpmQuery(query string, specfile string, container *lm_sdk_tools.LMTargetContainer) (string, error) {
	//query information from the specfile
	envVars := []string{
		"LC_ALL=C",
	}

	command := fmt.Sprintf("rpmspec -q --srpm --qf \"%s\" %s",
		query,
		specfile,
	)

	output, exitCode, err := lm_sdk_tools.RunInContainerOuput(container, false, envVars, command)
	if err != nil {
		return "", fmt.Errorf("Failed to query information from the spec file")
	}
	if exitCode != 0 {
		return "", fmt.Errorf("Failed to query information from the spec file\n%s", output[1])
	}

	return output[0], nil
}

/**
 * Tries to guess the tarball name from the given specfile. The assumption is
 * that SOURCE0 will always contain the tarball name. However SOURCE0 can contain variables
 * so we try to resolve those with running rpmspec
 *
 * @TODO: first try to run rpmspec to query SOURCES, if its only one value we can use that
 */
func (c *rpmbuildCmd) guessTarballName(specfile string, container *lm_sdk_tools.LMTargetContainer) (string, error) {

	file, err := os.Open(specfile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	source0Matcher, err := regexp.Compile("^Source0:\\s+(.*)")
	if err != nil {
		return "", fmt.Errorf("Error compiling the regexp: %v", err)
	}

	tarballName := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if !source0Matcher.MatchString(line) {
			continue
		}

		//we found the Source0 line, lets try to parse all variables

		varMatcher, err := regexp.Compile("%{([^}]*)}")
		if err != nil {
			return "", fmt.Errorf("Error compiling the regexp: %v", err)
		}

		tarballName = source0Matcher.FindStringSubmatch(line)[1]
		rpmVars := varMatcher.FindAllStringSubmatch(line, -1)
		for _, rpmVar := range rpmVars {
			output, err := c.rpmQuery(fmt.Sprintf("%%{%s}", rpmVar[1]), specfile, container)
			if err != nil {
				return "", err
			}
			tarballName = strings.Replace(tarballName, rpmVar[0], output, -1)
		}
		break
	}

	return tarballName, nil
}

func (c *rpmbuildCmd) findFilesByExt(dir string, extension string) []string {
	files := make([]string, 0, 10)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == extension {
			fmt.Printf("Found file: %s\n", path)
			files = append(files, path)
		}
		return nil
	})
	return files
}

func (c *rpmbuildCmd) selectIndexFromList(message string, list []string) int {

	fmt.Printf("\n%s\n", message)
	for idx, entry := range list {
		fmt.Printf("%d: %s\n", idx, entry)
	}

	selectedIndex := int(0)
	for true {
		fmt.Printf("Enter Nr (-1 to cancel): ")
		_, err := fmt.Scanf("%d", &selectedIndex)
		if err != nil || selectedIndex < -1 || selectedIndex > len(list)-1 {
			fmt.Fprintf(os.Stderr, "Please enter a valid number\n")
			continue
		}
		//if we reach this place a valid index was selected
		return selectedIndex
	}
	return -1
}

//This struct represents a element in the "zypper -x wp <builddep>" output
type solvable struct {
	Status  string `xml:"status,attr"`
	Name    string `xml:"name,attr"`
	Summary string `xml:"summary,attr"`
	Kind    string `xml:"kind,attr"`
}

func (c *rpmbuildCmd) installBuildDependencies(specfile string, container *lm_sdk_tools.LMTargetContainer) error {
	//query information from the specfile
	envVars := []string{
		"LC_ALL=C",
	}

	command := fmt.Sprintf("rpmspec -q --srpm --buildrequires %s",
		specfile,
	)

	fmt.Printf("\n----- Checking build dependencies -----\n")

	output, exitCode, err := lm_sdk_tools.RunInContainerOuput(container, false, envVars, command)
	if err != nil || exitCode != 0 {
		return fmt.Errorf("Failed to query build dependencies from the spec file. %s\n", output[1])
	}

	//the list of packages we need to install
	var packages []string
	for _, buildreq := range strings.Split(output[0], "\n") {

		if len(buildreq) == 0 {
			continue
		}

		zyppComm := fmt.Sprintf("zypper -x wp \"%s\"", buildreq)
		output, exitCode, err := lm_sdk_tools.RunInContainerOuput(container, false, envVars, zyppComm)
		if err != nil || exitCode != 0 {
			return fmt.Errorf("Failed to query zypper for the provider of: %s. %s\n", buildreq, output[1])
		}

		b := bytes.NewBufferString(output[0])

		alreadyProvided := false
		decoder := xml.NewDecoder(b)
		var notInstalledProviders []string
		for {
			// Read tokens from the XML document in a stream.
			t, _ := decoder.Token()
			if t == nil {
				break
			}

			// Inspect the type of the token just read.
			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "solvable" {
					var solvableEntry solvable
					err = decoder.DecodeElement(&solvableEntry, &se)
					if err != nil {
						return fmt.Errorf("Failed to parse zypper output: %v", err)
					}

					if solvableEntry.Kind == "package" {
						if solvableEntry.Status != "installed" {
							notInstalledProviders = append(notInstalledProviders, solvableEntry.Name)
						} else if solvableEntry.Status == "installed" {
							//if this build req is already installed we do not need to care anymore
							alreadyProvided = true
						}
					}
				}
			default:
			}

			//the buildreq is already provided by a installed package
			if alreadyProvided {
				break
			}
		}

		if !alreadyProvided {
			fmt.Printf("* %s needs to be installed.\n", buildreq)
			switch {
			case len(notInstalledProviders) == 0:
				continue
			case len(notInstalledProviders) == 1:
				packages = append(packages, notInstalledProviders[0])
			case len(notInstalledProviders) > 1:
				selectedIndex := c.selectIndexFromList(
					fmt.Sprintf("%s is provided by multiple packages, please select the package to install:", buildreq),
					notInstalledProviders,
				)
				if selectedIndex < 0 {
					return fmt.Errorf("Cancelled by user")
				}
				packages = append(packages, notInstalledProviders[selectedIndex])
			}
		} else {
			fmt.Printf("* %s already installed.\n", buildreq)
		}
	}

	if len(packages) > 0 {
		output, exitCode, err := lm_sdk_tools.RunInContainerOuput(
			container,
			true,
			envVars,
			fmt.Sprintf("zypper --non-interactive install %s", strings.Join(packages, " ")),
		)
		if err != nil || exitCode != 0 {
			return fmt.Errorf("Failed to install packages.\n%s\n", output[1])
		}
	}
	return nil
}

func (c *rpmbuildCmd) run(args []string) error {

	if len(args) < 2 {
		fmt.Fprint(os.Stderr, c.usage())
		os.Exit(1)
	}

	c.container = args[0]
	c.projectDir = args[1]

	//make sure the container is up and running
	container, err := lm_sdk_tools.LoadLMContainer(c.container)
	if err != nil {
		return fmt.Errorf("Could not connect to the Container: %v\n", err)
	}

	err = lm_sdk_tools.BootContainerSync(container)
	if err != nil {
		return err
	}

	if c.upgrade {
		me, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read path of lmsdk-target binary: %v\n", err)
			return err
		}

		comm := exec.Command(me, "upgrade", c.container)
		comm.Stdout = os.Stdout
		comm.Stderr = os.Stderr

		err = comm.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Upgrading the target failed.\n")
			return err
		}
	}

	//check if the project directory exists
	_, err = os.Stat(c.projectDir)
	if err != nil {
		return err
	}

	if len(c.specfile) == 0 {
		//search for the specfile
		specfiles := c.findFilesByExt(c.projectDir, ".spec")

		if len(specfiles) == 0 {
			return fmt.Errorf("No spec file found, cancelling build")
		} else if len(specfiles) > 1 {
			selectedIndex := c.selectIndexFromList(
				"Multiple spec files found, please select the specfile you want to use.",
				specfiles,
			)
			if selectedIndex < 0 {
				return fmt.Errorf("Cancelled by user")
			}
			c.specfile = specfiles[selectedIndex]
		} else {
			c.specfile = specfiles[0]
		}
	}

	fmt.Printf("Building using the specfile: %s\n", c.specfile)

	//create a clean build environment
	builddir, err := ioutil.TempDir("", "lmsdk-target")
	if err != nil {
		return err
	}
	//defer os.RemoveAll(builddir) // clean up

	fmt.Printf("Build dir: %s\n", builddir)

	rpmSourcesDir := filepath.Join(builddir, "SOURCES")
	err = os.MkdirAll(rpmSourcesDir, 0755)
	if err != nil {
		return err
	}

	//move the spec file to the build directory, so its available in the container
	_, err = exec.Command("cp", c.specfile, rpmSourcesDir).Output()
	if err != nil {
		return err
	}

	tarball := ""
	specfileName := filepath.Base(c.specfile)
	if len(c.tarballName) == 0 {
		fmt.Printf("There was no tarball prefix defined with -p, trying to guess from specfile.\n")
		tarball, err = c.guessTarballName(filepath.Join(rpmSourcesDir, specfileName), container)
		if err != nil {
			fmt.Printf("Failed to guess the tarball name, please try to pass a prefix with -p\nError: %v", err)
		}
	} else {
		tarball = c.tarballName
	}

	fmt.Printf("Creating the tarball: %s\n", tarball)

	//reaching this place we figured out how the source tarball is named
	//next step, copy over all rpm related files to the build directory

	//check if there is a known subdir that contains rpm files
	specialDirs := []string{
		"rpm",
		"skytree",
	}
	for _, specialDir := range specialDirs {
		_, err = os.Stat(filepath.Join(c.projectDir, specialDir))
		if err == nil {
			fmt.Printf("Copying: %s\n", filepath.Join(c.projectDir, specialDir, "*"))
			commOut, err := exec.Command("bash", "-c", fmt.Sprintf("cp -r %s %s", filepath.Join(c.projectDir, specialDir, "*"), rpmSourcesDir)).Output()
			if err != nil {
				return fmt.Errorf("Failed to copy the special dir: %s, %v\n%s", specialDir, err, commOut)
			}
		}
	}

	//copy over all patchfiles to the build directory
	//@TODO read extra files from spec and copy them
	patchfiles := c.findFilesByExt(c.projectDir, ".patch")
	for _, patchfile := range patchfiles {
		_, err = exec.Command("cp", patchfile, rpmSourcesDir).Output()
		if err != nil {
			return err
		}
	}

	//now create the source tarball
	tarballdir, err := ioutil.TempDir("", "lmsdk-target")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tarballdir) // clean up

	//rpmbuild expects the source directory to be named like %{Name}-%{Version}
	sourceDirName, err := c.rpmQuery("%{Name}-%{Version}", filepath.Join(rpmSourcesDir, specfileName), container)
	if err != nil {
		return fmt.Errorf("Failed to query the rpmspec for the source dir name :%v", err)
	}

	_, err = exec.Command("cp", "-r", c.projectDir, filepath.Join(tarballdir, sourceDirName)).Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("tar", "-C", tarballdir, "-ScpJf", filepath.Join(rpmSourcesDir, tarball), sourceDirName).Output()
	if err != nil {
		return err
	}

	//install build depenencies
	if c.installDeps {
		err = c.installBuildDependencies(filepath.Join(rpmSourcesDir, specfileName), container)
		if err != nil {
			return err
		}
	}

	//now finally build the packages
	envVars := []string{
		"LC_ALL=C",
		fmt.Sprintf("MAKEFLAGS=-j %d", c.jobs),
	}

	command := fmt.Sprintf("rpmbuild -bb %s --define \"_topdir %s\" --target %s",
		filepath.Join(rpmSourcesDir, specfileName),
		builddir,
		container.Architecture,
	)

	exitCode, err := lm_sdk_tools.RunInContainer(container, false, envVars, command, os.Stdout.Fd(), os.Stderr.Fd())
	if err != nil {
		return fmt.Errorf("Failed to execute rpmbuild command in the container: %v", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("The rpmbuild command failed")
	}

	if len(c.outputDirectory) > 0 {
		if _, err = os.Stat(c.outputDirectory); err != nil {
			fmt.Fprintf(os.Stderr, "Can not access output directory.\n")
			return err
		}

		fmt.Printf("Copying results from to %s\n", c.outputDirectory)
		command = fmt.Sprintf("cp -r %s %s", path.Join(builddir, "RPMS", "*"), c.outputDirectory)
		_, err = exec.Command("bash", "-c", command).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to copy results.\n")
			return err
		}
	}

	return nil
}
