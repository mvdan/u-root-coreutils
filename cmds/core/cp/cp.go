// Copyright 2016-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/mvdan/u-root-coreutils/pkg/cp"
)

func main() {
	runParams := cp.RunParams{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	os.Exit(cp.RunMain(runParams, os.Args[1:]...))
}
