package main

import (
	"context"
	"fmt"
	"os"
	"time"

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

	flag.StringVar(&controllerConfig.HcloudAPIToken, "hcloud-api-token", "", "Hetzner cloud API token")
	flag.IntVar(&controllerConfig.LeaseDuration, "lease-duration", 15, "Time to wait (in seconds) until next leader check")
	flag.IntVar(&controllerConfig.LeaseRenewDeadline, "lease-renew-deadline", 10, "Time to wait (in seconds) until next leader check")
	flag.StringVar(&controllerConfig.LeaseName, "lease-name", "fip", "Name of the lease lock for leaderelection")
	flag.StringVar(&controllerConfig.Namespace, "namespace", "", "Kubernetes Namespace")
	flag.StringVar(&controllerConfig.NodeName, "node-name", "", "Kubernetes Node name")
	flag.StringVar(&controllerConfig.PodName, "pod-name", "", "Kubernetes pod name")
	flag.StringVar(&controllerConfig.LogLevel, "log-level", "Info", "Log level")
	flag.StringVar(&controllerConfig.FloatingIPLabelSelector, "floating-ip-label-selector", "", "Selector for Floating IPs")
	flag.StringVar(&controllerConfig.NodeLabelSelector, "node-label-selector", "", "Selector for Nodes")
	flag.StringVar(&controllerConfig.PodLabelSelector, "pod-label-selector", "", "Selector for Pods. Should be the same key as specified in deployment")
	flag.DurationVar(&controllerConfig.BackoffDuration, "backoff-duration", time.Second, "Duration for first backoff")
	flag.Float64Var(&controllerConfig.BackoffFactor, "backoff-factor", 1.2, "Factor for backoff increase")
	flag.IntVar(&controllerConfig.BackoffSteps, "backoff-steps", 5, "Number of backoff retries")
	flag.StringVar(&controllerConfig.HealthCheckAddress, "health-check-address", ":8080", "Address the health and readiness endpoints listen on")
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

	healthServer := fipcontroller.NewHealthServer(controllerConfig.HealthCheckAddress, controller.Logger)
	go func() {
		if err := healthServer.Run(ctx); err != nil {
			controller.Logger.Errorf("health server stopped: %v", err)
		}
	}()
	// Clients are initialised and the controller is about to join leader
	// election, so it is ready to serve traffic.
	healthServer.SetReady(true)

	controller.RunWithLeaderElection(ctx)
}
