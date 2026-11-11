//go:build !(linux || darwin || windows) || android || ios

package blupcloud

// machineID is empty where the program can't read the device's id itself:
// on Android and iOS the app gives it (SetDevice).
func machineID() string { return "" }
