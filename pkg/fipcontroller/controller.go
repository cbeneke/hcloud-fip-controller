package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
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
	HcloudApiToken    string           `json:"hcloud_api_token,omitempty"`
	HcloudFloatingIPs stringArrayFlags `json:"hcloud_floating_ips,omitempty"`
	LeaseDuration     int              `json:"lease_duration,omitempty"`
	LeaseName         string           `json:"lease_name,omitempty"`
	Namespace         string           `json:"namespace,omitempty"`
	NodeAddressType   string           `json:"node_address_type,omitempty"`
	NodeName          string           `json:"node_name,omitempty"`
	PodName           string           `json:"pod_name,omitempty"`
	LogLevel          string           `json:"log_level,omitempty"`
}

type Controller struct {
	HetznerClient    *hcloud.Client
	KubernetesClient *kubernetes.Clientset
	Configuration    *Configuration
	Logger           *logrus.Logger
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

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	logger.SetReportCaller(true)
	logger.SetOutput(os.Stdout)

	loglevel, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("could not parse log level: %v", err)
	}
	logger.SetLevel(loglevel)

	return &Controller{
		HetznerClient:    hetznerClient,
		KubernetesClient: kubernetesClient,
		Configuration:    config,
		Logger:           logger,
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
	var undefinedErrs []string

	if configuration.HcloudApiToken == "" {
		undefinedErrs = append(errs, "hetzner cloud API token")
	}
	if len(configuration.HcloudFloatingIPs) <= 0 {
		undefinedErrs = append(errs, "hetzner cloud floating IPs")
	}
	if configuration.NodeName == "" {
		undefinedErrs = append(errs, "kubernetes node name")
	}
	if configuration.Namespace == "" {
		undefinedErrs = append(errs, "kubernetes namespace")
	}
	if configuration.LeaseDuration <= 0 {
		errs = append(errs, "lease duration needs to be greater than one")
	}

	if len(undefinedErrs) > 0 {
		errs = append(errs, fmt.Sprintf("required configuration options not configured: %s", strings.Join(errs, ", ")))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}
	return nil
}

func (controller *Controller) Run(ctx context.Context) error {
	if err := controller.UpdateFloatingIPs(ctx); err != nil {
		return err
	}
	controller.Logger.Info("Initialization complete. Starting reconciliation")

	for {
		select {
		case <-ctx.Done():
			controller.Logger.Info("Context Done. Shutting down")
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
			controller.Logger.Infof("Switching address '%s' to server '%s'", floatingIP.IP.String(), server.Name)
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
