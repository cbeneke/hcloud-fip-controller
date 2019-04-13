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
	Token   string
	Address string
}

type Client struct {
	Client        *hcloud.Client
	FloatingIP	  *hcloud.FloatingIP
	Configuration Configuration
}

func NewClient(ctx context.Context) (*Client, error) {
	client := Client{}

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
	client.Configuration = config

	client.Client = hcloud.NewClient(hcloud.WithToken(config.Token))

	client.FloatingIP, err = GetFipFromClient(ctx, &client)
	if err != nil {
		return nil, fmt.Errorf("could not get floating IP: %v", err)
	}

	return &client, nil
}

func GetFipFromClient(ctx context.Context, client *Client) (ip *hcloud.FloatingIP, err error) {
	ips, err := client.Client.FloatingIP.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch floating IPs: %v", err)
	}

	for _, ip := range ips {
		if ip.IP.Equal(net.ParseIP(client.Configuration.Address)) {
			return ip, nil
		}
	}

	return nil, fmt.Errorf("IP address %s not allocated", client.Configuration.Address)
}

func Run(client Client) error {
	fmt.Printf("Using IP address %s.\n", client.FloatingIP.IP.String())
	return nil
}
