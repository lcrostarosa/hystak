//go:build windows

package launch

import (
	"os"
	"os/signal"
)

var signalChan chan os.Signal

// ignoreSignals causes the parent to ignore interrupt signals
// while a child process is running.
func ignoreSignals() {
	signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
}

// restoreSignals restores default signal handling.
func restoreSignals() {
	signal.Stop(signalChan)
	signal.Reset(os.Interrupt)
}
