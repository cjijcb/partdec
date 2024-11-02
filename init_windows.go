//go:build windows

package partdec

import (
	"golang.org/x/sys/windows"
	"os"
)

func init() {
	var originalMode uint32

	stdout := windows.Handle(os.Stdout.Fd())

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
