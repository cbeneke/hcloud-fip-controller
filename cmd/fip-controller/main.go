package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/internal/app/fipcontroller"
	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
)

func main() {
	controllerConfig := &configuration.Configuration{}

	controllerConfig.ParseFlags()

	controller, err := fipcontroller.NewController(controllerConfig)
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise controller: %v", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller.RunWithLeaderElection(ctx)
}
