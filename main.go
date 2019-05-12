package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
)

func main() {
	// overwrite existing configs from ENV vars if present
	if apiToken := os.Getenv("HETZNER_CLOUD_API_TOKEN"); apiToken != "" {
		_ = flag.Set("hcloud-api-token", apiToken)
	}
	if floatingIP := os.Getenv("HETZNER_CLOUD_FLOATING_IP"); floatingIP != "" {
		_ = flag.Set("floating-ip", floatingIP)
	}
	if nodeName := os.Getenv("KUBERNETES_NODE_NAME"); nodeName != "" {
		_ = flag.Set("node-name", nodeName)
	}
	if nodeAddressType := os.Getenv("KUBERNETES_NODE_ADDRESS_TYPE"); nodeAddressType != "" {
		_ = flag.Set("node-address-type", nodeAddressType)
	}

	controllerConfig, err := fipcontroller.NewControllerConfiguration()
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise controllerConfig: %v", err))
		os.Exit(1)
	}

	controller, err := fipcontroller.NewController(controllerConfig)
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise controller: %v", err))
		os.Exit(1)
	}

	// TODO: Use channel with interrupt signal that blocks until it receives to cancel the context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = controller.Run(ctx)
	if err != nil {
		fmt.Printf("could not run controller: %v\n", err)
		os.Exit(1)
	}
}
