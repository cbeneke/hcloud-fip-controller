package fipcontroller

import (
	"context"
	"fmt"
	"net"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

func newHetznerClient(token string) (*hcloud.Client, error) {
	hetznerClient := hcloud.NewClient(hcloud.WithToken(token))
	return hetznerClient, nil
}

/*
 * Search and return the hcloud floatingIP object for a given string representation of a IPv4 or IPv6 address
 */
func (controller *Controller) floatingIP(ctx context.Context, ipAddress string) (ip *hcloud.FloatingIP, err error) {
	ips, err := controller.HetznerClient.FloatingIP.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch floating IPs: %v", err)
	}
	controller.Logger.Debugf("Fetched %d IP addresses", len(ips))

	for _, ip := range ips {
		if ip.Type == hcloud.FloatingIPTypeIPv4 && ip.IP.Equal(net.ParseIP(ipAddress)) {
			return ip, nil
		}
		if ip.Type == hcloud.FloatingIPTypeIPv6 && ip.Network.Contains(net.ParseIP(ipAddress)) {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("IP address '%s' not allocated", ipAddress)
}

/*
 * Search and return the hcloud Server objects for a given list of IP addresses.
 *  The IP Addresses can be public IPv4, IPv6 addresses or private addresses attached to any private network interface
 */
func (controller *Controller) servers(ctx context.Context, ips []net.IP) (serverList []*hcloud.Server, err error) {
	// Fetch all hetzner servers
	servers, err := controller.HetznerClient.Server.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch servers: %v", err)
	}
	controller.Logger.Debugf("Fetched %d servers", len(servers))

	for _, ip := range ips {
		for _, server := range servers {
			// IP must not be a floating IP, but might be private or public depending on the cluster configuration
			if server.PublicNet.IPv4.IP.Equal(ip) || server.PublicNet.IPv6.IP.Equal(ip) {
				controller.Logger.Debugf("Found matching public IP on server '%s'", server.Name)
				serverList = append(serverList, server)
				break
			}

			privateNet := searchPrivateNet(server.PrivateNet, ip)
			if privateNet != "" {
				controller.Logger.Debugf("Found matching private IP on network '%s' for server '%s'", privateNet, server.Name)
				serverList = append(serverList, server)
				break
			}

			return nil, fmt.Errorf("Could not find an IP for server '%s'", server.Name)
		}
	}
	return serverList, nil
}

/*
 * Search for a specified ip in the given privateNet of a server.
 * Return the network name if a network has been found and an empty string otherwise
 */
func searchPrivateNet(privateNet []hcloud.ServerPrivateNet, ip net.IP) string {
	for _, privateNet := range privateNet {
		if privateNet.IP.Equal(ip) {
			return privateNet.Network.Name
		}
	}
	return ""
}
