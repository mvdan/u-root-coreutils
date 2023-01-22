// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race
// +build !race

package gpio

import (
	"testing"

	"github.com/mvdan/u-root-coreutils/pkg/qemu"
	"github.com/mvdan/u-root-coreutils/pkg/vmtest"
)

func TestIntegration(t *testing.T) {
	vmtest.GolangTest(t, []string{"github.com/mvdan/u-root-coreutils/pkg/gpio"}, &vmtest.Options{
		QEMUOpts: qemu.Options{
			// Make GPIOs nums 10 to 20 available through the
			// mockup driver.
			KernelArgs: "gpio-mockup.gpio_mockup_ranges=10,20",
		},
	})
}
