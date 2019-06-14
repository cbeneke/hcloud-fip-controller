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
	HcloudApiToken   string
	HcloudFloatingIP string
	NodeAddressType  string
	NodeName         string
}

type Controller struct {
	HetznerClient    *hcloud.Client
	KubernetesClient *kubernetes.Clientset
	Configuration    *Configuration
}

func NewController(config *Configuration) (*Controller, error) {
	hetznerClient, err := hetznerClient(config.HcloudApiToken)
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
		Configuration:    config,
	}, nil
}

func ParseConfigFile(configuration *Configuration) error {
	if _, err := os.Stat("config/config.json"); err == nil {
		file, err := ioutil.ReadFile("config/config.json")
		if err != nil {
			return fmt.Errorf("failed to read config: %v", err)
		}
		err = json.Unmarshal(file, &configuration)
		if err != nil {
			return fmt.Errorf("failed to decode config: %v", err)
		}
	}

	return  nil
}

func ValidateControllerConfig(configuration *Configuration) error {
	// Validate required configs
	var errs []string

	if configuration.HcloudApiToken == "" {
		errs = append(errs, "hetzner cloud API token")
	}
	if configuration.HcloudFloatingIP == "" {
		errs = append(errs, "hetzner cloud floating IP")
	}
	if configuration.NodeName == "" {
		errs = append(errs, "kubernetes node name")
	}
	if len(errs) > 0 {
		return  fmt.Errorf("required configuration options not configured: %s", strings.Join(errs, ", "))
	}
	return nil
}

func (controller *Controller) Run(ctx context.Context) error {
	if err := controller.UpdateFloatingIP(ctx); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(30 * time.Second):
			if err := controller.UpdateFloatingIP(ctx); err != nil {
				return err
			}
		}
	}
}

func (controller *Controller) UpdateFloatingIP(ctx context.Context) error {
	nodeAddress, err := controller.nodeAddress(controller.Configuration.NodeName, controller.Configuration.NodeAddressType)
	if err != nil {
		return fmt.Errorf("could not get kubernetes node address: %v", err)
	}
	server, err := controller.server(ctx, nodeAddress)
	if err != nil {
		return fmt.Errorf("could not get configured server: %v", err)
	}
	floatingIP, err := controller.floatingIP(ctx, controller.Configuration.HcloudFloatingIP)
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
