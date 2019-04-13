package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
)

func main() {
	ctx := context.Background()
	client, err := fipcontroller.NewClient(ctx)
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise client: %v", err))
		os.Exit(1)
	}

	err = fipcontroller.Run(*client)
	if err != nil {
		fmt.Println(fmt.Errorf("could not run client: %v", err))
		os.Exit(1)
	}
}