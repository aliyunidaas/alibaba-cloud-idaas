package utils

import (
	"fmt"
	"net"
	"runtime"
	"strings"
)

func GetMacAddress() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getMacAddressForMacOs()
	case "linux":
		return getMacAddressForLinux()
	case "windows":
		return getMacAddressForWindows()
	default:
		return "", fmt.Errorf("unsupported OS：%s", runtime.GOOS)
	}
}

func getMacAddressForMacOs() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Name == "en0" {
			if iface.HardwareAddr.String() != "" {
				return iface.HardwareAddr.String(), nil
			}
		}
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 &&
			iface.Flags&net.FlagUp != 0 &&
			iface.HardwareAddr.String() != "" {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no-mac-found")
}

func getMacAddressForLinux() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Name == "eth0" {
			if iface.HardwareAddr.String() != "" {
				return iface.HardwareAddr.String(), nil
			}
		}
	}

	priorityNames := []string{"ens33", "ens192", "enp0s3", "enp0s25"}

	for _, name := range priorityNames {
		for _, iface := range interfaces {
			if iface.Name == name {
				if iface.HardwareAddr.String() != "" {
					return iface.HardwareAddr.String(), nil
				}
			}
		}
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 &&
			iface.Flags&net.FlagUp != 0 &&
			iface.HardwareAddr.String() != "" &&
			!isVirtualInterface(iface.Name) {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no-mac-found")
}

func getMacAddressForWindows() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	priorityNames := []string{"Ethernet", "以太网", "WLAN", "Wi-Fi", "无线网络连接"}

	for _, name := range priorityNames {
		for _, iface := range interfaces {
			if iface.Name == name || contains(iface.Name, name) {
				if iface.HardwareAddr.String() != "" {
					return iface.HardwareAddr.String(), nil
				}
			}
		}
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 &&
			iface.Flags&net.FlagUp != 0 &&
			iface.HardwareAddr.String() != "" &&
			!isVirtualInterface(iface.Name) {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no-mac-found")
}

func isVirtualInterface(name string) bool {
	virtualPrefixes := []string{
		"vmnet", "vboxnet", "virbr", "veth", "docker",
		"br-", "vEthernet", "Hyper-V", "Loopback",
	}

	for _, prefix := range virtualPrefixes {
		if contains(name, prefix) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
