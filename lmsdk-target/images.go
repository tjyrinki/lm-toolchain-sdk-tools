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
	"os"
	"strings"
	"time"

	"link-motion.com/lm-toolchain-sdk-tools"
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
	return `Shows the available Link-Motion SDK images.

lmsdk-target images`
}

func (c *imagesCmd) flags() {
}

func findRelevantImages() ([]imageDesc, error) {

	client := &http.Client{}

	url := "https://sdk.link-motion.com/images/meta/1.0/index-user"
	if len(os.Getenv(lm_sdk_tools.LmImageServerEnvVar)) > 0 {
		serverName := os.Getenv(lm_sdk_tools.LmImageServerEnvVar)
		url = fmt.Sprintf("%s/meta/1.0/index-user", serverName)
	}

	req, err := http.NewRequest("GET", url, nil)

	if len(os.Getenv("LM_USERNAME")) > 0 {
		req.SetBasicAuth(
			os.Getenv("LM_USERNAME"),
			os.Getenv("LM_PASSWORD"),
		)
	}

	resp, err := client.Do(req)
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
