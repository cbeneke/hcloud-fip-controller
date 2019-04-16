package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const Version = "0.0.1"

type Configuration struct {
	Token   string
	Address string
}

type Client struct {
	HetznerClient *hcloud.Client
	KubeClient    *kubernetes.Clientset
	Configuration Configuration
	NodeName      string
}

func NewClient() (*Client, error) {
	// TODO: Move config reading out of NewClient() and pass as struct
	file, err := ioutil.ReadFile("config/config.json")
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}

	var config Configuration
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config: %v", err)
	}

	hetznerClient := hcloud.NewClient(hcloud.WithToken(config.Token))

	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get kubeconfig: %v", err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not get kubernetes client: %v", err)
	}

	return &Client{
		HetznerClient: hetznerClient,
		KubeClient:    kubeClient,
		Configuration: config,
		NodeName:      os.Getenv("NODE_NAME"),
	}, nil
}

func (client *Client) Run(ctx context.Context) error {
	for {
		select {
		case <-time.After(30 * time.Second):
			err := client.UpdateFloatingIPs(ctx)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (client *Client) UpdateFloatingIPs(ctx context.Context) error {
	nodeAddress, err := client.nodeAddress()
	if err != nil {
		return fmt.Errorf("could not get kubernetes node address: %v", err)
	}

	server, err := client.server(ctx, nodeAddress)
	if err != nil {
		return fmt.Errorf("could not get current serverAddress: %v", err)
	}

	floatingIP, err := client.floatingIP(ctx)
	if err != nil {
		return err
	}

	if server.ID != floatingIP.Server.ID {
		fmt.Printf("Switching address '%s' to server '%s'.\n", floatingIP.IP.String(), server.Name)
		_, response, err := client.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
		if err != nil {
			return fmt.Errorf("could not update floating IP: %v", err)
		}
		if response.StatusCode != 201 {
			return fmt.Errorf("could not update floating IP: Got HTTP Code %d, expected 201", response.StatusCode)
		}
	} else {
		fmt.Printf("Address %s already assigned to server '%s'. Nothing to do.\n", floatingIP.IP.String(), server.Name)
	}

	return nil
}

func (client *Client) floatingIP(ctx context.Context) (ip *hcloud.FloatingIP, err error) {
	ips, err := client.HetznerClient.FloatingIP.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch floating IPs: %v", err)
	}

	for _, ip := range ips {
		if ip.Type == hcloud.FloatingIPTypeIPv4 && ip.IP.Equal(net.ParseIP(client.Configuration.Address)) {
			return ip, nil
		}
		if ip.Type == hcloud.FloatingIPTypeIPv6 && ip.Network.Contains(net.ParseIP(client.Configuration.Address)) {
			return ip, nil
		}
	}

	// TODO: Try to return with the address and no error
	return nil, fmt.Errorf("IP address %s not allocated", client.Configuration.Address)
}

func (client *Client) server(ctx context.Context, ip net.IP) (server *hcloud.Server, err error) {
	servers, err := client.HetznerClient.Server.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch servers: %v", err)
	}

	for _, server := range servers {
		if server.PublicNet.IPv4.IP.Equal(ip) {
			return server, nil
		}
		if server.PublicNet.IPv6.IP.Equal(ip) {
			return server, nil
		}
	}

	// TODO: Try to return with the address and no error
	return nil, fmt.Errorf("no server with IP address %s found", ip.String())
}

func (client *Client) nodeAddress() (address net.IP, err error) {
	nodes, err := client.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list nodes: %v", err)
	}

	var addresses []corev1.NodeAddress
	for _, node := range nodes.Items {
		if node.Name == client.NodeName {
			addresses = node.Status.Addresses
			break
		}
	}

	for _, address := range addresses {
		// TODO: Make address.Type configurable
		if address.Type == corev1.NodeInternalIP {
			return net.ParseIP(address.Address), nil
		}
	}

	// TODO: Try to return with the address and no error
	return nil, fmt.Errorf("could not find address for current node")
}
