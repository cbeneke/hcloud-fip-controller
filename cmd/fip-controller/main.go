package main

import (
	"context"
	"fmt"
	"os"

	"github.com/namsral/flag"

	"github.com/cbeneke/hcloud-fip-controller/internal/app/fipcontroller"
	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
)

func main() {
	controllerConfig := &configuration.Configuration{}

	// Set defaults for flag.Var values
	controllerConfig.NodeAddressType = configuration.NodeAddressTypeExternal

	// Parse flags
	flag.Var(&controllerConfig.HcloudFloatingIPs, "hcloud-floating-ip", "Hetzner cloud floating IP Address. This option can be specified multiple times")
	flag.Var(&controllerConfig.NodeAddressType, "node-address-type", "Kubernetes node address type")

	flag.StringVar(&controllerConfig.HcloudApiToken, "hcloud-api-token", "", "Hetzner cloud API token")
	flag.IntVar(&controllerConfig.LeaseDuration, "lease-duration", 30, "Time to wait (in seconds) until next leader check")
	flag.StringVar(&controllerConfig.LeaseName, "lease-name", "fip", "Name of the lease lock for leaderelection")
	flag.StringVar(&controllerConfig.Namespace, "namespace", "", "Kubernetes Namespace")
	flag.StringVar(&controllerConfig.NodeName, "node-name", "", "Kubernetes Node name")
	flag.StringVar(&controllerConfig.PodName, "pod-name", "", "Kubernetes pod name")
	flag.StringVar(&controllerConfig.LogLevel, "log-level", "Info", "Log level")
	flag.StringVar(&controllerConfig.FloatingIPLabelSelector, "floating-ip-label-selector", "", "Selector for Floating IPs")
	flag.StringVar(&controllerConfig.NodeLabelSelector, "node-label-selector", "", "Selector for Nodes")

	// Parse options from file
	if _, err := os.Stat("config/config.json"); err == nil {
		if err := controllerConfig.VarsFromFile("config/config.json"); err != nil {
			fmt.Println(fmt.Errorf("could not parse controller config file: %v", err))
			os.Exit(1)
		}
	}

	// When default- and file-configs are read, parse command line options with highest priority
	flag.Parse()

	controller, err := fipcontroller.NewController(controllerConfig)
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise controller: %v", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller.RunWithLeaderElection(ctx)
}
