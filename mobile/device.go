package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The phone's id and name, written by the Android side when the app starts
// (MainActivity.writeDeviceInfo) in the app's own folder: Blup Cloud knows
// the same phone again after a reinstall, and the panel lists it by the
// name the person sees in the phone's settings (blupcloud.SetDevice).
var deviceOnce sync.Once

func loadDevice() {
	deviceOnce.Do(func() {
		// Empty outside Android (the app run on a computer to develop): the
		// computer's own id is used then.
		root := application.Mobile.StoragePath()
		if root == "" {
			return
		}
		b, err := os.ReadFile(filepath.Join(root, "device.json"))
		if err != nil {
			return
		}
		var d struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if json.Unmarshal(b, &d) == nil {
			blupcloud.SetDevice(d.ID, d.Name)
		}
	})
}
