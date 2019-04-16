package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Configuration struct {
	Token   string
	Address string
}

type Controller struct {
	HetznerClient    *hcloud.Client
	KubernetesClient *kubernetes.Clientset
	Configuration    *Configuration
	NodeName         string
}

func NewController(config *Configuration) (*Controller, error) {

	hetznerClient := hcloud.NewClient(hcloud.WithToken(config.Token))

	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get kubeconfig: %v", err)
	}

	kubernetesClient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not get kubernetes client: %v", err)
	}

	return &Controller{
		HetznerClient:    hetznerClient,
		KubernetesClient: kubernetesClient,
		Configuration:    config,
		NodeName:         os.Getenv("NODE_NAME"),
	}, nil
}

func ParseConfig() (*Configuration, error) {

	var config Configuration

	file, err := ioutil.ReadFile("config/config.json")
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config: %v", err)
	}

	return &config, nil
}

func (controller *Controller) Run(ctx context.Context) error {
	// TODO: use select{} with ctx.Done to gracefully shutdown
	for {
		nodeAddress, err := controller.nodeAddress()
		if err != nil {
			return fmt.Errorf("could not get kubernetes node address: %v", err)
		}

		server, err := controller.server(ctx, nodeAddress)
		if err != nil {
			return fmt.Errorf("could not get current serverAddress: %v", err)
		}

		floatingIP, err := controller.floatingIP(ctx)
		if err != nil {
			return err
		}

		if server.ID != floatingIP.Server.ID {
			fmt.Printf("Switching address '%s' to server '%s'.\n", floatingIP.IP.String(), server.Name)
			_, response, err := controller.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
			if err != nil {
				return fmt.Errorf("could not update floating IP: %v", err)
			}
			if response.StatusCode != 201 {
				return fmt.Errorf("could not update floating IP: Got HTTP Code %d, expected 201", response.StatusCode)
			}
		} else {
			fmt.Printf("Address %s already assigned to server '%s'. Nothing to do.\n", floatingIP.IP.String(), server.Name)
		}

		time.Sleep(30 * time.Second)
	}
}
