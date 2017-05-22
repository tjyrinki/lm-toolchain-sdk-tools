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
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type imageDesc struct {
	Distribution string    `json:"distribution"`
	Version      string    `json:"version"`
	Variant      string    `json:"variant"`
	Arch         string    `json:"arch"`
	UploadDate   time.Time `json:"uploadDate"`
}

type imagesCmd struct {
}

func (c *imagesCmd) usage() string {
	return `Shows the available Ubuntu SDK images.

lmsdk-target images`
}

func (c *imagesCmd) flags() {
}

func findRelevantImages() ([]imageDesc, error) {

	resp, err := http.Get("http://tre-ci.build.link-motion.com/sysroots/meta/1.0/index-user")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	reader := bufio.NewScanner(resp.Body)
	var imageDescs []imageDesc

	for reader.Scan() {
		line := reader.Text()
		fields := strings.Split(line, ";")
		if len(fields) != 7 {
			continue
		}

		//Mon Jan 2 15:04:05 MST 2006  (MST is GMT-0700)
		datetime, err := time.Parse("2006 01 02 15:04", fields[4])
		if err != nil {
			fmt.Printf("Failed to parse date: %v", err)
			continue
		}

		imageDescs = append(imageDescs, imageDesc{
			Distribution: fields[0],
			Version:      fields[1],
			Arch:         fields[2],
			Variant:      fields[3],
			UploadDate:   datetime,
		})

	}
	return imageDescs, nil
}

func (c *imagesCmd) run(args []string) error {

	imageDescs, err := findRelevantImages()
	if err != nil {
		return err
	}

	js, err := json.MarshalIndent(imageDescs, "  ", "  ")
	if err != nil {
		return fmt.Errorf("Error while formatting data from the server. error: %v,\n", err)
	}
	fmt.Printf("%s\n", js)
	return nil
}
