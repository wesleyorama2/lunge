//go:build !windows

package output

import (
	"os"

	"github.com/mattn/go-isatty"
)

// checkIsTerminal checks if the file is a terminal on Unix systems.
func checkIsTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
