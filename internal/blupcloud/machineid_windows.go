//go:build windows

package blupcloud

import "golang.org/x/sys/windows/registry"

// machineID is Windows' MachineGuid, made when Windows is installed. The
// 64-bit view of the registry, also from a 32-bit build.
func machineID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer k.Close()
	id, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return ""
	}
	return id
}
