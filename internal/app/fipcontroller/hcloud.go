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

func (controller *Controller) floatingIP(ctx context.Context, ipAddress string) (ip *hcloud.FloatingIP, err error) {
	ips, err := controller.HetznerClient.FloatingIP.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch floating IPs: %v", err)
	}
	controller.Logger.Debugf("Fetched %s IP addresses", len(ips))

	for _, ip := range ips {
		if ip.Type == hcloud.FloatingIPTypeIPv4 && ip.IP.Equal(net.ParseIP(ipAddress)) {
			return ip, nil
		}
		if ip.Type == hcloud.FloatingIPTypeIPv6 && ip.Network.Contains(net.ParseIP(ipAddress)) {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("IP address %s not allocated", ipAddress)
}

func (controller *Controller) server(ctx context.Context, ip net.IP) (server *hcloud.Server, err error) {
	servers, err := controller.HetznerClient.Server.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch servers: %v", err)
	}
	controller.Logger.Debugf("Fetched %s servers", len(servers))

	for _, server := range servers {
		if server.PublicNet.IPv4.IP.Equal(ip) || server.PublicNet.IPv6.IP.Equal(ip) {
			return server, nil
		}
	}
	return nil, fmt.Errorf("no server with IP address %s found", ip.String())
}
