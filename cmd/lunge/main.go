// Copyright (c) 2025, Wesley Brown
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/wesleyorama2/lunge/internal/cli"
)

// Main is the entry point for the application
// It's exported to make it testable
func Main() int {
	if err := cli.Execute(); err != nil {
		return 1
	}
	return 0
}

func main() {
	os.Exit(Main())
}
