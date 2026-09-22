package spotify

import "strings"

// FindDevice gives exact opaque IDs priority over case-insensitive device names.
func FindDevice(devices []Device, selector string) (Device, bool) {
	if selector == "" {
		return Device{}, false
	}
	for _, device := range devices {
		if device.ID == selector {
			return device, true
		}
	}
	for _, device := range devices {
		if strings.EqualFold(device.Name, selector) {
			return device, true
		}
	}
	return Device{}, false
}
