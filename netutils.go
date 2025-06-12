package netutils

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"
	"syscall"
)

// Htons converts a short (16-bit) integer from host byte order to network byte order.
func Htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

// GetInterfaceIndex returns the index of the network interface with the given name.
func GetInterfaceIndex(iface string) int {
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		log.Fatalf("Failed to get interface index for %s: %v", iface, err)
	}

	return ifaceObj.Index
}

// OpenRawSocket opens raw socket by name and return its file descriptor
func OpenRawSocket(name string) (int, error) {
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(Htons(syscall.ETH_P_ALL)))
	if err != nil {
		return -1, fmt.Errorf("failed to open raw socket: %w", err)
	}

	ifaceName := [16]byte{}
	copy(ifaceName[:], name)

	sll := syscall.SockaddrLinklayer{
		Protocol: Htons(syscall.ETH_P_ALL),
		Ifindex:  GetInterfaceIndex(name),
	}

	if err := syscall.Bind(fd, &sll); err != nil {
		return -1, fmt.Errorf("failed to bind socket to interface %s: %w", name, err)
	}

	return fd, nil
}

// EnableIpv4Forwarding writes 1 to '/proc/sys/net/ipv4/ip_forward'
func EnableIpv4Forwarding() error {
	file, err := os.OpenFile("/proc/sys/net/ipv4/ip_forward", os.O_RDWR, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", "/proc/sys/net/ipv4/ip_forward", err)
	}

	defer file.Close()

	_, err = file.Write([]byte{'1'})
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// FindIPv4 returns first ipv4 in slice of addresses or "0.0.0.0"
func FindIPv4(addrs []net.Addr) netip.Addr {
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}

		if ipAddr := netip.MustParseAddr(ip.String()); ipAddr.Is4() {
			return ipAddr
		}
	}

	return netip.IPv4Unspecified()
}

type InterfaceInfo struct {
	Name     string
	HardAddr net.HardwareAddr
	IP       net.IP
}

// GetInterfaceInfo gather interface mac and ip address
func GetInterfaceInfo(name string) (InterfaceInfo, error) {
	i, err := net.InterfaceByName(name)
	if err != nil {
		return InterfaceInfo{}, fmt.Errorf("failed to get interface by name: %w", err)
	}

	addrs, err := i.Addrs()
	if err != nil {
		return InterfaceInfo{}, fmt.Errorf("failed to get addresses by interface %s: %w", name, err)
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return InterfaceInfo{}, fmt.Errorf("failed to parse cidr: %w", err)
		}

		if ip.To4() != nil {
			return InterfaceInfo{
				Name:     name,
				IP:       ip,
				HardAddr: i.HardwareAddr,
			}, nil
		}
	}

	return InterfaceInfo{
		HardAddr: i.HardwareAddr,
	}, nil
}

// GetDefaultGatewayInfo gather gateway info by interface name
func GetDefaultGatewayInfo(ifce string) (InterfaceInfo, error) {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return InterfaceInfo{}, fmt.Errorf("failed to open: %w", err)
	}

	splitted := strings.Split(string(data), "\n")

	for _, str := range splitted {
		if strings.Contains(str, ifce) {
			splittedBySpace := strings.Split(removeExtraSpaces(str), " ")

			mac, err := net.ParseMAC(splittedBySpace[3])
			if err != nil {
				return InterfaceInfo{}, fmt.Errorf("failed to parse mac: %w", err)
			}

			ip := net.ParseIP(splittedBySpace[0])
			if ip == nil {
				return InterfaceInfo{}, fmt.Errorf("failed to parse ip %s", splittedBySpace[0])
			}

			return InterfaceInfo{
				Name:     ifce,
				IP:       ip,
				HardAddr: mac,
			}, nil

		}
	}

	return InterfaceInfo{}, nil
}

func removeExtraSpaces(s string) string {
	var prevIsSpace bool

	n := make([]byte, 0, len(s))

	for _, v := range s {
		if prevIsSpace && v == ' ' {
			continue
		}

		if v == ' ' {
			prevIsSpace = true
		} else {
			prevIsSpace = false
		}

		n = append(n, byte(v))
	}

	return string(n)
}
