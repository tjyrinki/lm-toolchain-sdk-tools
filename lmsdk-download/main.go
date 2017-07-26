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
 * Authors:
 * Juhapekka Piiroinen <juhapekka.piiroinen@link-motion.com>
 * Benjamin Zeller <benjamin.zeller@link-motion.com>
 */
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func usage() {
	fmt.Println("----")
	fmt.Println(" OR USE ONE OR MORE:")
	fmt.Println("    export LM_USERNAME=user")
	fmt.Println("    export LM_PASSWORD=pass")
	fmt.Println("    export LM_FILENAME=sometarget.txt")
	fmt.Println("    export LM_SKIP_LICENCE=1")
	fmt.Println("    export LM_URL=https://127.0.0.1")
}

func main() {

	username := flag.String("username", os.Getenv("LM_USERNAME"), "username")
	password := flag.String("password", os.Getenv("LM_PASSWORD"), "password")
	filename := flag.String("filename", os.Getenv("LM_FILENAME"), "filename")
	checkCerts := flag.Bool("no-cert-check", os.Getenv("LM_SKIP_LICENCE") == "1", "Disable certificate checks when connecting to https")
	url := flag.String("url", os.Getenv("LM_URL"), "url")

	flag.Parse()

	if *filename == "" {
		flag.Usage()
		usage()
		os.Exit(1)
	}
	if *url == "" {
		flag.Usage()
		usage()
		os.Exit(1)
	}

	transCfg := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: *checkCerts}, // ignore expired SSL certificates
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}

	client := &http.Client{
		Transport: transCfg,
	}
	req, err := http.NewRequest("GET", *url, nil)
	if err != nil {
		// handle err
		fmt.Println(err)
		os.Exit(1)
	}

	if len(*username) > 0 && len(*password) > 0 {
		req.SetBasicAuth(*username, *password)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "Server returned status code: %s\n", resp.Status)
		os.Exit(1)
	}

	fp, err := os.OpenFile(*filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create output file: %v\n", err)
		os.Exit(1)
	}
	defer fp.Close() // defer close

	if _, err := io.Copy(fp, resp.Body); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to write output file: %v\n", err)
	}
}
