package main

import (
	"fmt"
	"net"
)

// NetworkInterface 系统网络接口信息
type NetworkInterface struct {
	Name       string   `json:"name"`
	IP         string   `json:"ip"`
	IsUp       bool     `json:"is_up"`
	IsLoopback bool     `json:"is_loopback"`
	Hardware   string   `json:"hardware"`
	Addrs      []string `json:"addrs"`
}

// ListInterfaces 列出系统所有网络接口
func ListInterfaces() ([]NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var result []NetworkInterface
	for _, iface := range ifaces {
		ni := NetworkInterface{
			Name:       iface.Name,
			IsUp:       iface.Flags&net.FlagUp != 0,
			IsLoopback: iface.Flags&net.FlagLoopback != 0,
			Hardware:   iface.HardwareAddr.String(),
		}

		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ni.Addrs = append(ni.Addrs, addr.String())
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip != nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
					if ni.IP == "" {
						ni.IP = ip.String()
					}
				}
			}
		}

		result = append(result, ni)
	}

	return result, nil
}
