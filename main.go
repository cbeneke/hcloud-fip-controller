package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
)

func main() {
	controllerConfig, err := fipcontroller.NewControllerConfiguration()
	if err != nil {
		fmt.Println(fmt.Errorf("could not parse controllerConfig: %v", err))
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

	err = controller.Run(ctx, controllerConfig)
	if err != nil {
		fmt.Printf("could not run controller: %v\n", err)
		os.Exit(1)
	}
}
