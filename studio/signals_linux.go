package main

import "os"

// WebKitGTK's JavaScript engine uses signal 10 (SIGUSR1) for its garbage
// collector by default, which the Go runtime handles too, and says so in
// the terminal ("Overriding existing handler for signal 10"). It takes
// another one from JSC_SIGNAL_FOR_GC: 12 (SIGUSR2), which Go leaves alone.
// Set before the window (and WebKit) starts.
func init() {
	if os.Getenv("JSC_SIGNAL_FOR_GC") == "" {
		os.Setenv("JSC_SIGNAL_FOR_GC", "12")
	}
}
