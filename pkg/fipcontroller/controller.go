package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"

	"k8s.io/client-go/kubernetes"
)

type Configuration struct {
	HetznerAPIToken   string
	FloatingIPAddress string
	NodeAddressType   string
	NodeName          string
}

type Controller struct {
	HetznerClient    *hcloud.Client
	KubernetesClient *kubernetes.Clientset
}

func NewController(config *Configuration) (*Controller, error) {
	hetznerClient, err := hetznerClient(config.HetznerAPIToken)
	if err != nil {
		return nil, fmt.Errorf("could not initialise hetzner client: %v", err)
	}

	kubernetesClient, err := kubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("could not initialise kubernetes client: %v", err)
	}

	return &Controller{
		HetznerClient:    hetznerClient,
		KubernetesClient: kubernetesClient,
	}, nil
}

func NewControllerConfiguration() (*Configuration, error) {
	var config Configuration

	// Read config from file if present
	if _, err := os.Stat("config/config.json"); err == nil {
		file, err := ioutil.ReadFile("config/config.json")
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %v", err)
		}
		err = json.Unmarshal(file, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to decode config: %v", err)
		}
	}

	// overwrite existing configs from ENV vars if present
	if apiToken := os.Getenv("HETZNER_CLOUD_API_TOKEN"); apiToken != "" {
		config.HetznerAPIToken = apiToken
	}
	if floatingIP := os.Getenv("HETZNER_CLOUD_FLOATING_IP"); floatingIP != "" {
		config.FloatingIPAddress = floatingIP
	}
	if nodeName := os.Getenv("KUBERNETES_NODE_NAME"); nodeName != "" {
		config.NodeName = nodeName
	}
	if nodeAddressType := os.Getenv("KUBERNETES_NODE_ADDRESS_TYPE"); nodeAddressType != "" {
		config.NodeAddressType = nodeAddressType
	}

	// Use defaults for unset optional configs
	if config.NodeAddressType == "" {
		config.NodeAddressType = "external"
	}

	// Validate required configs
	var errs []string

	if config.HetznerAPIToken == "" {
		errs = append(errs, "hetzner cloud API token required but not configured")
	}
	if config.FloatingIPAddress == "" {
		errs = append(errs, "hetzner floating IP required but not configured")
	}
	if config.NodeName == "" {
		errs = append(errs, "kubernetes node name required but not configured")
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("controller configuration invalid: %s", strings.Join(errs, ","))
	}

	return &config, nil
}

func (controller *Controller) Run(ctx context.Context, config *Configuration) error {
	if err := controller.UpdateFloatingIP(ctx, config); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(30 * time.Second):
			if err := controller.UpdateFloatingIP(ctx, config); err != nil {
				return err
			}
		}
	}
}

func (controller *Controller) UpdateFloatingIP(ctx context.Context, config *Configuration) error {
	nodeAddress, err := controller.nodeAddress(config.NodeName, config.NodeAddressType)
	if err != nil {
		return fmt.Errorf("could not get kubernetes node address: %v", err)
	}
	server, err := controller.server(ctx, nodeAddress)
	if err != nil {
		return fmt.Errorf("could not get configured server: %v", err)
	}
	floatingIP, err := controller.floatingIP(ctx, config.FloatingIPAddress)
	if err != nil {
		return fmt.Errorf("could not get configured floating IP: %v", err)
	}

	if floatingIP.Server == nil || server.ID != floatingIP.Server.ID {
		fmt.Printf("Switching address '%s' to server '%s'.\n", floatingIP.IP.String(), server.Name)
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
