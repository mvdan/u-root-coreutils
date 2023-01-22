// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race
// +build !race

package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mvdan/u-root-coreutils/pkg/qemu"
	"github.com/mvdan/u-root-coreutils/pkg/uroot"
	"github.com/mvdan/u-root-coreutils/pkg/vmtest"
)

// testPkgs returns a slice of tests to run.
func testPkgs(t *testing.T) []string {
	// Packages which do not contain tests (or do not contain tests for the
	// build target) will still compile a test binary which vacuously pass.
	cmd := exec.Command("go", "list",
		"github.com/mvdan/u-root-coreutils/cmds/boot/...",
		"github.com/mvdan/u-root-coreutils/cmds/core/...",
		"github.com/mvdan/u-root-coreutils/cmds/exp/...",
		"github.com/mvdan/u-root-coreutils/pkg/...",
	)
	cmd.Env = append(os.Environ(), "GOARCH="+vmtest.TestArch())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	pkgs := strings.Fields(strings.TrimSpace(string(out)))

	// TODO: Some tests do not run properly in QEMU at the moment. They are
	// blocklisted. These tests fail for mostly two reasons:
	// 1. either it requires networking (not enabled in the kernel)
	// 2. or it depends on some test files (for example /bin/sleep)
	blocklist := []string{
		"github.com/mvdan/u-root-coreutils/cmds/core/cmp",
		"github.com/mvdan/u-root-coreutils/cmds/core/dd",
		"github.com/mvdan/u-root-coreutils/cmds/core/fusermount",
		"github.com/mvdan/u-root-coreutils/cmds/core/gosh",
		"github.com/mvdan/u-root-coreutils/cmds/core/wget",
		"github.com/mvdan/u-root-coreutils/cmds/core/which",
		// Some of TestEdCommands do not exit properly and end up left running. No idea how to fix this yet.
		"github.com/mvdan/u-root-coreutils/cmds/exp/ed",
		"github.com/mvdan/u-root-coreutils/cmds/exp/pox",
		"github.com/mvdan/u-root-coreutils/pkg/crypto",
		"github.com/mvdan/u-root-coreutils/pkg/tarutil",
		"github.com/mvdan/u-root-coreutils/pkg/ldd",

		// These have special configuration.
		"github.com/mvdan/u-root-coreutils/pkg/gpio",
		"github.com/mvdan/u-root-coreutils/pkg/mount",
		"github.com/mvdan/u-root-coreutils/pkg/mount/block",
		"github.com/mvdan/u-root-coreutils/pkg/mount/loop",
		"github.com/mvdan/u-root-coreutils/pkg/ipmi",
		"github.com/mvdan/u-root-coreutils/pkg/smbios",

		// Missing xzcat in VM.
		"github.com/mvdan/u-root-coreutils/cmds/exp/bzimage",
		"github.com/mvdan/u-root-coreutils/pkg/boot/bzimage",

		// No Go compiler in VM.
		"github.com/mvdan/u-root-coreutils/pkg/uroot",
		"github.com/mvdan/u-root-coreutils/pkg/uroot/builder",

		// ??
		"github.com/mvdan/u-root-coreutils/pkg/tss",
		"github.com/mvdan/u-root-coreutils/pkg/syscallfilter",
	}
	if vmtest.TestArch() == "arm64" {
		blocklist = append(
			blocklist,
			"github.com/mvdan/u-root-coreutils/pkg/strace",

			// These tests run in 1-2 seconds on x86, but run
			// beyond their huge timeout under arm64 in the VM. Not
			// sure why. Slow emulation?
			"github.com/mvdan/u-root-coreutils/cmds/core/pci",
			"github.com/mvdan/u-root-coreutils/cmds/exp/cbmem",
			"github.com/mvdan/u-root-coreutils/pkg/vfile",
		)
	}
	for i := 0; i < len(pkgs); i++ {
		for _, b := range blocklist {
			if pkgs[i] == b {
				pkgs = append(pkgs[:i], pkgs[i+1:]...)
			}
		}
	}

	return pkgs
}

// TestGoTest effectively runs "go test ./..." inside a QEMU instance. The
// tests run as root and can do all sorts of things not possible otherwise.
func TestGoTest(t *testing.T) {
	pkgs := testPkgs(t)

	o := &vmtest.Options{
		QEMUOpts: qemu.Options{
			Timeout: 120 * time.Second,
			Devices: []qemu.Device{
				// Bump this up so that some unit tests can happily
				// and questionably pre-claim large bytes slices.
				//
				// e.g. pkg/mount/gpt/gpt_test.go need to claim 4.29G
				//
				//     disk = make([]byte, 0x100000000)
				qemu.ArbitraryArgs{"-m", "6G"},

				// aarch64 VMs start at 1970-01-01 without RTC explicitly set.
				qemu.ArbitraryArgs{"-rtc", "base=localtime,clock=vm"},
			},
		},
		BuildOpts: uroot.Opts{
			ExtraFiles: []string{
				"/etc/group",
				"/etc/passwd",
			},
		},
	}
	vmtest.GolangTest(t, pkgs, o)
}
