package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
)

func main() {
	client, err := fipcontroller.NewClient()
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise client: %v", err))
		os.Exit(1)
	}

	// TODO: Use channel with interrupt signal that blocks until it receives to cancel the context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = client.Run(ctx)
	if err != nil {
		fmt.Printf("could not run client: %v\n", err)
		os.Exit(1)
	}
}
