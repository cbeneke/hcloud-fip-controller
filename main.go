package hcloud_fip_controller

import (
	"context"
	"fmt"
	"os"

	"github.com/cbeneke/hcloud-fip-controller/pkg/fipcontroller"
)

func main() {
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	client, err := fipcontroller.NewClient()
	if err != nil {
		fmt.Println(fmt.Errorf("could not initialise client: %v", err))
		os.Exit(1)
	}

	err = fipcontroller.Run(ctx, client)
	if err != nil {
		fmt.Println(fmt.Errorf("could not run client: %v", err))
		os.Exit(1)
	}
}
