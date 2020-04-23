package fipcontroller

import (
	"context"
	"fmt"
	"os"
	"sort"
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
	nodeAddressList, err := controller.nodeAddressList(controller.Configuration.NodeAddressType)
	if err != nil {
		return fmt.Errorf("could not get addressList for active kubernetes Nodes")
	}

	runningServers, err := controller.servers(ctx, nodeAddressList)
	if err != nil {
		return fmt.Errorf("Could not get server objects for addressList")
	}

	// Sort servers by number of assigned public ips
	sort.Slice(runningServers, func(i, j int) bool {
		return len(runningServers[i].PublicNet.FloatingIPs) < len(runningServers[j].PublicNet.FloatingIPs)
	})

	// Get floatingIPs from config if specified, otherwise from hetzner api
	floatingIPs, err := controller.getFloatingIPs(ctx)
	if err != nil {
		return fmt.Errorf("Could not get floatingIPs: %v", err)
	}
	// Next server to apply a floating ip to
	currentServer := 0

	for _, floatingIP := range floatingIPs {
		controller.Logger.Debugf("Checking floating IP: %s", floatingIP.IP.String())

		// (Re)assign floatingIP if no server is assigned or the assigned server is not running
		// Since we already have all running server in a slice we can just search through it
		if floatingIP.Server == nil || !findServerByID(runningServers, floatingIP.Server) {
			var server *hcloud.Server
			server = runningServers[currentServer]

			controller.Logger.Infof("Switching address '%s' to server '%s'", floatingIP.IP.String(), server.Name)
			_, response, err := controller.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
			if err != nil {
				return fmt.Errorf("could not update floating IP '%s': %v", floatingIP.IP.String(), err)
			}
			if response.StatusCode != 201 {
				return fmt.Errorf("could not update floating IP '%s': Got HTTP Code %d, expected 201", floatingIP.IP.String(), response.StatusCode)
			}
			currentServer = (currentServer + 1) % len(runningServers)
		}

	}
	return nil
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

/*
 * Fetches all floatingIPs from hetzner api with optional label selector.
 * For backwards compatibility this still uses hardcoded ips if specified in config
 */
func (controller *Controller) getFloatingIPs(ctx context.Context) ([]*hcloud.FloatingIP, error) {
	// Use hardcoded ips if specified
	// TODO fetch FloatingIPs once beforehand (maybe????)
	if len(controller.Configuration.HcloudFloatingIPs) > 0 {
		floatingIPs := []*hcloud.FloatingIP{}
		for _, floatingIPAddr := range controller.Configuration.HcloudFloatingIPs {
			floatingIP, err := controller.floatingIP(ctx, floatingIPAddr)
			if err != nil {
				return nil, fmt.Errorf("could not get floating IP '%s': %v", floatingIPAddr, err)
			}
			floatingIPs = append(floatingIPs, floatingIP)
		}
		return floatingIPs, nil
	}

	// Fetch ips from hetzner api with optional LabelSelector
	floatingIPListOpts := hcloud.FloatingIPListOpts{}
	if controller.Configuration.FloatingIPsLabelSelector != "" {
		listOpts := hcloud.ListOpts{}
		listOpts.LabelSelector = controller.Configuration.FloatingIPsLabelSelector
		floatingIPListOpts = hcloud.FloatingIPListOpts{ListOpts: listOpts}
	}

	floatingIPs, err := controller.HetznerClient.FloatingIP.AllWithOpts(ctx, floatingIPListOpts)
	if err != nil {
		return floatingIPs, fmt.Errorf("could not get floating IPs: %v", err)
	}
	return floatingIPs, nil
}
