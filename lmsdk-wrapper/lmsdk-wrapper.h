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
#ifndef LMSDK_WRAPPER_H_INCLUDED
#define LMSDK_WRAPPER_H_INCLUDED

//returns true if the child terminated normally, that is, by calling exit(3) or _exit(2), or by returning from main().
int get_WIFEXITED(int status);

/*
returns the exit status of the child. 
This consists of the least significant 8 bits of the status argument that the child specified in a call to exit(3) 
or _exit(2) or as the argument for a return statement in main(). This macro should only be employed if WIFEXITED returned true.
*/
int get_WEXITSTATUS(int status);

//returns true if the child process was terminated by a signal.
int get_WIFSIGNALED(int status);

/*
returns the number of the signal that caused the child process to terminate. 
This macro should only be employed if WIFSIGNALED returned true.*/
int get_WTERMSIG(int status);

int get_WCOREDUMP(int status);


#endif