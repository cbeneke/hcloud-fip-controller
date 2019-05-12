package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
	"github.com/go-kit/kit/log"
)

func main() {
	logger := log.With(log.NewJSONLogger(log.NewSyncWriter(os.Stdout)), "time", log.DefaultTimestampUTC, "level", "INFO")

	controllerConfig, err := fipcontroller.NewControllerConfiguration()
	if err != nil {
		_ = logger.Log("msg", fmt.Errorf("could not parse controllerConfig: %v", err))
		os.Exit(1)
	}

	controller, err := fipcontroller.NewController(controllerConfig, logger)
	if err != nil {
		_ = logger.Log("msg", fmt.Errorf("could not initialise controller: %v", err))
		os.Exit(1)
	}

	// TODO: Use channel with interrupt signal that blocks until it receives to cancel the context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = controller.Run(ctx)
	if err != nil {
		_ = logger.Log("msg", fmt.Errorf("could not run controller: %v", err))
		os.Exit(1)
	}
}
