package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/client-go/kubernetes"
)

type Configuration struct {
	HetznerAPIToken   string
	FloatingIPAddress string
	NodeAddressType   string
}

type Controller struct {
	HetznerClient    *hcloud.Client
	KubernetesClient *kubernetes.Clientset
	Configuration    *Configuration
	NodeName         string
	Logger           log.Logger
}

func NewController(config *Configuration, logger log.Logger) (*Controller, error) {
	hetznerClient, err := hetznerClient(config.HetznerAPIToken)
	if err != nil {
		return nil, fmt.Errorf("could not initialise kubernetes client: %v", err)
	}

	kubernetesClient, err := kubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("could not initialise kubernetes client: %v", err)
	}

	return &Controller{
		HetznerClient:    hetznerClient,
		KubernetesClient: kubernetesClient,
		Configuration:    config,
		NodeName:         os.Getenv("NODE_NAME"),
		Logger:           logger,
	}, nil
}

func NewControllerConfiguration() (*Configuration, error) {
	var config Configuration

	file, err := ioutil.ReadFile("config/config.json")
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config: %v", err)
	}

	if config.HetznerAPIToken == "" {
		token := os.Getenv("HETZNER_API_TOKEN")
		if token == "" {
			return nil, fmt.Errorf("hetzner API token required but not configured")
		}
		config.HetznerAPIToken = token
	}

	if config.FloatingIPAddress == "" {
		return nil, fmt.Errorf("floating IP required but not configured")
	}

	switch config.NodeAddressType {
	case "":
		config.NodeAddressType = "external"
	case "external":
		config.NodeAddressType = "external"
	case "internal":
		config.NodeAddressType = "internal"
	default:
		return nil, fmt.Errorf("nodeAddressType configured with '%s' but only '', 'external' or 'internal' allowed", config.NodeAddressType)
	}

	return &config, nil
}

func (controller *Controller) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(30 * time.Second):
			err := controller.UpdateFloatingIPs(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (controller *Controller) UpdateFloatingIPs(ctx context.Context) error {
	nodeAddress, err := controller.nodeAddress()
	if err != nil {
		return fmt.Errorf("could not get kubernetes node address: %v", err)
	}

	server, err := controller.server(ctx, nodeAddress)
	if err != nil {
		return fmt.Errorf("could not get current serverAddress: %v", err)
	}

	floatingIP, err := controller.floatingIP(ctx)
	if err != nil {
		return err
	}

	if floatingIP.Server == nil || server.ID != floatingIP.Server.ID {
		_ = controller.Logger.Log("msg", "Switching address '"+floatingIP.IP.String()+"' to server '"+server.Name+"'.")
		_, response, err := controller.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
		if err != nil {
			return fmt.Errorf("could not update floating IP: %v", err)
		}
		if response.StatusCode != 201 {
			return fmt.Errorf("could not update floating IP: Got HTTP Code %d, expected 201", response.StatusCode)
		}
	}
	return nil
}
