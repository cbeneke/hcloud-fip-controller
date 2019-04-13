package fipcontroller

import (
	"encoding/json"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"os"
)


type Configuration struct {
	Token string
	FloatingIP string
}

type Client struct {
	Config Configuration
	HetznerClient *hcloud.Client
}

func newClient() (*Client, error) {
	client := Client{}

	client.Config = Configuration{}
	file, err := os.Open("config.json")
	if err != nil {
		return nil, fmt.Errorf("Could not open config file: %v", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&client.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not decode config: %v", err)
	}

	client.HetznerClient = hcloud.NewClient(hcloud.WithToken(client.Config.Token))

	return &client, nil
}