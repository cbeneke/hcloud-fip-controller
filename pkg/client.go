package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"net"
	"os"
)

type Configuration struct {
	Token      string
	FloatingIP string
}

type Client struct {
	Client        *hcloud.Client
	Configuration Configuration
}

func newClient() (*Client, error) {
	config := Configuration{}
	file, err := os.Open("config.json")
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config: %v", err)
	}

	hClient := hcloud.NewClient(hcloud.WithToken(config.Token))

	return &Client{hClient, config}, nil
}

func getFipID(ctx context.Context, fip string, client *Client) (id int, err error) {
	ips, err := client.Client.FloatingIP.All(ctx)
	if err != nil {
		return -1, fmt.Errorf("could not fetch floating IPs: %v", err)
	}

	for _, ip := range ips {
		if ip.IP.Equal(net.ParseIP(fip)) {
			return ip.ID, nil
		}
	}

	return -1, fmt.Errorf("IP address %s not allocated", fip)
}

func run() error {
	ctx := context.Background()
	client, err := newClient()
	if err != nil {
		return fmt.Errorf("could not initialise hetzner client: %v", err)
	}
	fip, err := getFipID(ctx, client.Configuration.FloatingIP, client)
	if err != nil {
		return fmt.Errorf("could not get floating IP: %v", err)
	}

	fmt.Printf("Using IP address %d.\n", fip)
	return nil
}
