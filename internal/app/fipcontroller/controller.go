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
	// Validate controller config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("controller config invalid: %v", err)
	}

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
 * Main logic function.
 *  Searches for running hetzner cloud servers and sort them by fewest assigned floating ips.
 *  It then (re)assigns all unassigned ips or ips that are assigned to non running servers to the sorted running serves.
 *
 */
func (controller *Controller) UpdateFloatingIPs(ctx context.Context) error {
	controller.Logger.Debugf("Checking floating IPs")

	// Get running servers for floating ip assignment
	nodeAddressList, err := controller.nodeAddressList(ctx, controller.Configuration.NodeAddressType)
	if err != nil {
		return fmt.Errorf("could not get addressList for active kubernetes nodes: %v", err)
	}

	runningServers, err := controller.servers(ctx, nodeAddressList)
	if err != nil {
		return fmt.Errorf("Could not get server objects for addressList: %v", err)
	}

	// Get floatingIPs from config if specified, otherwise from hetzner api
	floatingIPs, err := controller.getFloatingIPs(ctx)
	if err != nil {
		return fmt.Errorf("Could not get floatingIPs: %v", err)
	}

	for _, floatingIP := range floatingIPs {
		controller.Logger.Debugf("Checking floating IP: %s", floatingIP.IP.String())

		// (Re)assign floatingIP if no server is assigned or the assigned server is not running
		// Since we already have all running server in a slice we can just search through it
		if floatingIP.Server == nil || !findServerByID(runningServers, floatingIP.Server) {
			// Get the server with the lowest amount of fips (cant be nil since we know that servers can't be empty)
			server := findServerWithLowestFIP(runningServers)

			controller.Logger.Infof("Switching address '%s' to server '%s'", floatingIP.IP.String(), server.Name)
			_, response, err := controller.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
			if err != nil {
				return fmt.Errorf("could not update floating IP '%s': %v", floatingIP.IP.String(), err)
			}
			if response.StatusCode != 201 {
				return fmt.Errorf("could not update floating IP '%s': Got HTTP Code %d, expected 201", floatingIP.IP.String(), response.StatusCode)
			}
			// Add placeholder floating ip to server so that findServerWithLowestFIP will always get a correct server
			server.PublicNet.FloatingIPs = append(server.PublicNet.FloatingIPs, &hcloud.FloatingIP{})
		}

	}
	return nil
}

/*
 * Find the server with the lowest amount of floating IPs
 */
func findServerWithLowestFIP(servers []*hcloud.Server) *hcloud.Server {
	if len(servers) < 1 {
		return nil
	}
	server := servers[0]
	for _, s := range servers {
		if len(s.PublicNet.FloatingIPs) < len(server.PublicNet.FloatingIPs) {
			server = s
		}
	}
	return server
}

/*
 * Find a server in a slice by its id
 * Returns a fully filled server struct if a server was found
 */
func findServerByID(slice []*hcloud.Server, val *hcloud.Server) bool {
	for _, item := range slice {
		if item.ID == val.ID {
			return true
		}
	}
	return false
}
