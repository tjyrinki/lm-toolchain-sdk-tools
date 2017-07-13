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
#include "lmsdk-wrapper.h"

#include <sys/types.h>
#include <sys/wait.h>

int get_WIFEXITED(int status) {
    return WIFEXITED(status);
}

int get_WEXITSTATUS(int status) {
    return WEXITSTATUS(status);
}

int get_WIFSIGNALED(int status) {
    return WIFSIGNALED(status);
}

int get_WTERMSIG(int status) {
    return WTERMSIG(status);
}

int get_WCOREDUMP(int status) {
    return WCOREDUMP(status);
}