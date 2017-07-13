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
 * Ported from shellescape Python library: https://github.com/chrissimpkins/shellescape
 * Author: Chris Simpkins
 */
package lm_sdk_tools

import (
	"regexp"
	"strings"
)

var _find_unsafe = regexp.MustCompile("[^\\w@%+=:,./-]")

func QuoteString(s string) string {
	//Return a shell-escaped version of the string *s*.
	if len(s) == 0 {
		return "''"
	}

	if !_find_unsafe.MatchString(s) {
		return s
	}

	// use single quotes, and put single quotes into double quotes
	// the string $'b is then quoted as '$'"'"'b'

	return "'" + strings.Replace(s, "'", "'\"'\"'", -1) + "'"
}
