package netutils

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateTapDevice create tap device, sets cidr to it and up via ip command
func CreateTapDevice(name, cidr string) error {
	_, err := exec.Command("/sbin/ip", "tuntap", "add", "dev", name, "mode", "tap").Output()
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	_, err = exec.Command("/sbin/ip", "addr", "add", cidr, "dev", name).Output()
	if err != nil {
		return fmt.Errorf("failed to add address to device: %w", err)
	}

	_, err = exec.Command("/sbin/ip", "link", "set", name, "up").Output()
	if err != nil {
		return fmt.Errorf("failed to up device: %w", err)
	}

	return nil
}

// StopTapDevice stop tap device by name via ip command
func StopTapDevice(name string) error {
	_, err := exec.Command("/sbin/ip", "tuntap", "del", "dev", name, "mode", "tap").Output()
	if err != nil {
		return fmt.Errorf("failed to stop device: %w", err)
	}

	return nil
}

// GetDefaultGateway parses command 'ip route' and return default device name
func GetDefaultGatewayDevice() string {
	data, err := exec.Command("/sbin/ip", "route").Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "default") {
			splitted := strings.Split(line, " ")

			for i, v := range splitted {
				if v == "dev" {
					return splitted[i+1]
				}
			}
		}
	}

	return ""
}
