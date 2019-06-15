package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"

	"k8s.io/client-go/kubernetes"
)

type stringArrayFlags []string

func (flags *stringArrayFlags) String() string {
	return fmt.Sprintf("['%s']", strings.Join(*flags, "', '"))
}
func (flags *stringArrayFlags) Set(value string) error {
	*flags = append(*flags, value)
	return nil
}

type Configuration struct {
	HcloudApiToken    string
	HcloudFloatingIPs stringArrayFlags
	NodeAddressType   string
	NodeName          string
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

// Read given config file and overwrite options from given Configuration
func (configuration *Configuration) VarsFromFile(configFile string) error {
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = json.Unmarshal(file, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config file: %v", err)
	}

	return nil
}

func (configuration *Configuration) Validate() error {
	var errs []string

	if configuration.HcloudApiToken == "" {
		errs = append(errs, "hetzner cloud API token")
	}
	if len(configuration.HcloudFloatingIPs) <= 0 {
		errs = append(errs, "hetzner cloud floating IPs")
	}
	if configuration.NodeName == "" {
		errs = append(errs, "kubernetes node name")
	}
	if len(errs) > 0 {
		return fmt.Errorf("required configuration options not configured: %s", strings.Join(errs, ", "))
	}
	return nil
}

func (controller *Controller) Run(ctx context.Context) error {
	if err := controller.UpdateFloatingIPs(ctx); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(30 * time.Second):
			if err := controller.UpdateFloatingIPs(ctx); err != nil {
				return err
			}
		}
	}
}

func (controller *Controller) UpdateFloatingIPs(ctx context.Context) error {
	nodeAddress, err := controller.nodeAddress(controller.Configuration.NodeName, controller.Configuration.NodeAddressType)
	if err != nil {
		return fmt.Errorf("could not get kubernetes node address: %v", err)
	}
	server, err := controller.server(ctx, nodeAddress)
	if err != nil {
		return fmt.Errorf("could not get configured server: %v", err)
	}

	for _, floatingIPAddr := range controller.Configuration.HcloudFloatingIPs {
		floatingIP, err := controller.floatingIP(ctx, floatingIPAddr)
		if err != nil {
			return fmt.Errorf("could not get floating IP '%s': %v", floatingIPAddr, err)
		}

		if floatingIP.Server == nil || server.ID != floatingIP.Server.ID {
			fmt.Printf("Switching address '%s' to server '%s'.\n", floatingIP.IP.String(), server.Name)
			_, response, err := controller.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
			if err != nil {
				return fmt.Errorf("could not update floating IP '%s': %v", floatingIP.IP.String(), err)
			}
			if response.StatusCode != 201 {
				return fmt.Errorf("could not update floating IP '%s': Got HTTP Code %d, expected 201", floatingIP.IP.String(), response.StatusCode)
			}
		}
	}

	return nil
}
