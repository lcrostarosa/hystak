//go:build !windows

package launch

import (
	"os"
	"os/signal"
	"syscall"
)

var signalChan chan os.Signal

// ignoreSignals causes the parent to ignore SIGINT and SIGTERM
// while a child process is running.
func ignoreSignals() {
	signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
}

// restoreSignals restores default signal handling.
func restoreSignals() {
	signal.Stop(signalChan)
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)
}
