package fipcontroller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
)

type Controller struct {
	HetznerClient    *hcloud.Client
	KubernetesClient *kubernetes.Clientset
	Configuration    *configuration.Configuration
	Logger           *logrus.Logger
}

func NewController(config *configuration.Configuration) (*Controller, error) {
	hetznerClient, err := newHetznerClient(config.HcloudApiToken)
	if err != nil {
		return nil, fmt.Errorf("could not initialise hetzner client: %v", err)
	}

	kubernetesClient, err := newKubernetesClient()
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

/*
 * Main threat.
 *  Run update IP function once initially and every 30s afterwards
 */
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

/*
 * Main logical function.
 *  Searches for the Hetzner Cloud Node object the pod is running on and validates that all configured floating IPs
 *  are attached to that node.
 */
func (controller *Controller) UpdateFloatingIPs(ctx context.Context) error {
	controller.Logger.Debugf("Checking floating IPs")

	nodeAddress, err := controller.nodeAddress(controller.Configuration.NodeName, controller.Configuration.NodeAddressType)
	if err != nil {
		return fmt.Errorf("could not get kubernetes node address: %v", err)
	}
	controller.Logger.Debugf("Found node address: %s", nodeAddress.String())

	server, err := controller.server(ctx, nodeAddress)
	if err != nil {
		return fmt.Errorf("could not get configured server: %v", err)
	}
	controller.Logger.Debugf("Found server: %s (%d)", server.Name, server.ID)

	for _, floatingIPAddr := range controller.Configuration.HcloudFloatingIPs {
		floatingIP, err := controller.floatingIP(ctx, floatingIPAddr)
		if err != nil {
			return fmt.Errorf("could not get floating IP '%s': %v", floatingIPAddr, err)
		}
		controller.Logger.Debugf("Checking floating IP: %s", floatingIP.IP.String())

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
