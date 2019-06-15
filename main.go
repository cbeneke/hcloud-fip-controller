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
	flag.StringVar(&controllerConfig.HcloudApiToken, "hcloud-api-token", "", "Hetzner cloud API token")
	flag.StringVar(&controllerConfig.HcloudFloatingIP, "hcloud-floating-ip", "", "Hetzner cloud floating IP Address")
	flag.StringVar(&controllerConfig.NodeName, "node-name", "", "Kubernetes Node name")
	flag.StringVar(&controllerConfig.NodeAddressType, "node-address-type", "external", "Kubernetes node address type")

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

	err = controller.Run(ctx)
	if err != nil {
		fmt.Printf("could not run controller: %v\n", err)
		os.Exit(1)
	}
}
