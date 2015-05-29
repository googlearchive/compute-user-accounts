// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logger provides a logger that writes to the syslog.
package logger

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime/debug"
)

var writer *syslog.Writer

func init() {
	// The default level for logging done in other packages is error.
	var err error
	writer, err = syslog.New(syslog.LOG_ERR|syslog.LOG_AUTH, "gcua")
	if err != nil {
		panic(err)
	}
	log.SetOutput(writer)
}

// Info logs an info level message to the syslog.
func Info(a ...interface{}) {
	writer.Info(fmt.Sprint(a...))
}

// Infof logs an info level message to the syslog.
func Infof(format string, a ...interface{}) {
	writer.Info(fmt.Sprintf(format, a...))
}

// Notice logs a notice level message to the syslog.
func Notice(a ...interface{}) {
	writer.Notice(fmt.Sprint(a...))
}

// Noticef logs a notice level message to the syslog.
func Noticef(format string, a ...interface{}) {
	writer.Notice(fmt.Sprintf(format, a...))
}

// Error logs an error level message to the syslog.
func Error(a ...interface{}) {
	writer.Err(fmt.Sprint(a...))
}

// Errorf logs an error level message to the syslog.
func Errorf(format string, a ...interface{}) {
	writer.Err(fmt.Sprintf(format, a...))
}

// Fatal logs a crit level message to the syslog and exits with a non-zero exit
// code.
func Fatal(a ...interface{}) {
	msg := fmt.Sprint(a...)
	writer.Crit(msg)
	fmt.Fprintln(os.Stderr, msg)
	debug.PrintStack()
	os.Exit(255)
}

// Fatalf logs a crit level message to the syslog and exits with a non-zero exit
// code.
func Fatalf(format string, a ...interface{}) {
	Fatal(fmt.Sprintf(format, a...))
}
