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
		// check if a server was found for the given ip, if not throw error
		server := controller.searchServerForIP(servers, ip)
		if server == nil {
			return nil, fmt.Errorf("Could not find an a server for ip '%s'", ip)
		}
		serverList = append(serverList, server)
	}
	return serverList, nil
}

/*
 * Search for a hetzner Server that has the given ip in any of its networks
 */
func (controller *Controller) searchServerForIP(servers []*hcloud.Server, ip net.IP) *hcloud.Server {
	for _, server := range servers {
		// IP must not be a floating IP, but might be private or public depending on the cluster configuration
		if server.PublicNet.IPv4.IP.Equal(ip) || server.PublicNet.IPv6.IP.Equal(ip) {
			controller.Logger.Debugf("Found matching public IP on server '%s'", server.Name)
			return server
		}

		hasPrivateNet := false
		for _, privateNet := range server.PrivateNet {
			if privateNet.IP.Equal(ip) {
				hasPrivateNet = true
				break
			}
		}

		if hasPrivateNet {
			controller.Logger.Debugf("Found matching private IP for server '%s'", server.Name)
			return server
		}

	}
	return nil
}

/*
 * Fetches all floatingIPs from hetzner api with optional label selector.
 * For backwards compatibility this still uses hardcoded ips if specified in config
 */
func (controller *Controller) getFloatingIPs(ctx context.Context) ([]*hcloud.FloatingIP, error) {
	// Use hardcoded ips if specified
	// TODO fetch FloatingIPs once beforehand (maybe????)
	if len(controller.Configuration.HcloudFloatingIPs) > 0 {
		floatingIPs := []*hcloud.FloatingIP{}
		for _, floatingIPAddr := range controller.Configuration.HcloudFloatingIPs {
			floatingIP, err := controller.floatingIP(ctx, floatingIPAddr)
			if err != nil {
				return nil, fmt.Errorf("could not get floating IP '%s': %v", floatingIPAddr, err)
			}
			floatingIPs = append(floatingIPs, floatingIP)
		}
		return floatingIPs, nil
	}

	// Fetch ips from hetzner api with optional LabelSelector
	floatingIPListOpts := hcloud.FloatingIPListOpts{}
	if controller.Configuration.FloatingIPLabelSelector != "" {
		listOpts := hcloud.ListOpts{}
		listOpts.LabelSelector = controller.Configuration.FloatingIPLabelSelector
		floatingIPListOpts = hcloud.FloatingIPListOpts{ListOpts: listOpts}
	}

	floatingIPs, err := controller.HetznerClient.FloatingIP.AllWithOpts(ctx, floatingIPListOpts)
	if err != nil {
		return nil, fmt.Errorf("could not get floating IPs: %v", err)
	}
	return floatingIPs, nil
}
