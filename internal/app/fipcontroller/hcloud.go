package fipcontroller

import (
	"context"
	"fmt"
	"k8s.io/client-go/util/retry"
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
	var ips []*hcloud.FloatingIP
	err = retry.OnError(controller.Backoff, alwaysRetry, func() error {
		ips, err = controller.HetznerClient.FloatingIP.All(ctx)
		return err
	})
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
 * Search and return the hcloud Server object for a given IP address.
 *  The IP Address can be a public IPv4, IPv6 address or a private address attached to any private network interface
 */
func (controller *Controller) server(ctx context.Context, ip net.IP) (server *hcloud.Server, err error) {
	var servers []*hcloud.Server
	err = retry.OnError(controller.Backoff, alwaysRetry, func() error {
		servers, err = controller.HetznerClient.Server.All(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch servers: %v", err)
	}
	controller.Logger.Debugf("Fetched %d servers", len(servers))

	for _, server := range servers {
		// IP must not be a floating IP, but might be private or public depending on the cluster configuration
		if server.PublicNet.IPv4.IP.Equal(ip) || server.PublicNet.IPv6.IP.Equal(ip) {
			controller.Logger.Debugf("Found matching public IP on server '%s'", server.Name)
			return server, nil
		}
		for _, privateNet := range server.PrivateNet {
			if privateNet.IP.Equal(ip) {
				controller.Logger.Debugf("Found matching private IP on network '%s' for server '%s'", privateNet.Network.Name, server.Name)
				return server, nil
			}
		}
	}
	return nil, fmt.Errorf("no server with IP address %s found", ip.String())
}
