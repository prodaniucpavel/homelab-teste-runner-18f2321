package misc

// index, mac, type (wifi/eth), ip, subnet, gw, mac, description

import (
	"net"
)

type ClientInterface struct {
	Index       int      `json:"interface_index"`
	Mac         string   `json:"mac"`
	Description string   `json:"description"`
	Addresses   []string `json:"addresses"`
}

func GetNetworkInterfaces() []ClientInterface {
	interfaces := make([]ClientInterface, 0)

	systemInterfaces, err := net.Interfaces()

	if err != nil {
		panic(err)
	}

	for _, systemInterface := range systemInterfaces {
		ipAddresses, err := systemInterface.Addrs()

		if err != nil {
			panic(err)
		}

		addresses := make([]string, 0)

		for _, ipAddress := range ipAddresses {
			addresses = append(addresses, ipAddress.String())
		}

		clientInterface := ClientInterface{
			Index:       systemInterface.Index,
			Mac:         systemInterface.HardwareAddr.String(),
			Description: systemInterface.Name,
			Addresses:   addresses,
		}

		interfaces = append(interfaces, clientInterface)
	}

	return interfaces
}
