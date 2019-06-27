package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
	"github.com/namsral/flag"
)

func main() {
	controllerConfig := &fipcontroller.Configuration{}

	// Setup flags
	flag.Var(&controllerConfig.HcloudFloatingIPs, "hcloud-floating-ip", "Hetzner cloud floating IP Address. This option can be specified multiple times")

	flag.StringVar(&controllerConfig.HcloudApiToken, "hcloud-api-token", "", "Hetzner cloud API token")
	flag.IntVar(&controllerConfig.LeaseDuration, "lease-duration", 30, "Time to wait (in seconds) until next leader check")
	flag.StringVar(&controllerConfig.LeaseName, "lease-name", "fip-lock", "Name of the lease lock for leaderelection")
	flag.StringVar(&controllerConfig.Namespace, "namespace", "", "Kubernetes Namespace")
	flag.StringVar(&controllerConfig.NodeAddressType, "node-address-type", "external", "Kubernetes node address type")
	flag.StringVar(&controllerConfig.NodeName, "node-name", "", "Kubernetes Node name")
	flag.StringVar(&controllerConfig.PodName, "pod-name", "", "Kubernetes pod name")

	// Parse options from file
	if _, err := os.Stat("config/config.json"); err == nil {
		if err := controllerConfig.VarsFromFile("config/config.json"); err != nil {
			fmt.Println(fmt.Errorf("could not parse controller config file: %v", err))
			os.Exit(1)
		}
	}

	// When default- and file-configs are read, parse command line options with highest priority
	flag.Parse()

	if err := controllerConfig.Validate(); err != nil {
		fmt.Println(fmt.Errorf("controllerConfig not valid: %v", err))
		os.Exit(1)
	}

	controller, err := fipcontroller.NewController(controllerConfig)
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise controller: %v", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller.RunWithLeaderElection(ctx)
}
