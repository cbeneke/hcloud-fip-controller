package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const Version = "0.0.1"

type Configuration struct {
	Token   string
	Address string
}

type Client struct {
	HetznerClient *hcloud.Client
	KubeClient    *kubernetes.Clientset
	Configuration Configuration
}

func NewClient() (*Client, error) {
	client := Client{}

	config := Configuration{}
	file, err := os.Open("config/config.json")
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config: %v", err)
	}
	client.Configuration = config

	client.HetznerClient = hcloud.NewClient(hcloud.WithToken(config.Token))
	if err != nil {
		return nil, fmt.Errorf("could not get floating IP: %v", err)
	}
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get kubeconfig: %v", err)
	}
	client.KubeClient, err = kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not get kubernetes client: %v", err)
	}

	return &client, nil
}

func (client *Client) Run(ctx context.Context) error {
	for {
		nodeAddress, err := client.nodeAddress()
		if err != nil {
			return fmt.Errorf("could not get kubernetes node address: %v", err)
		}

		serverAddress, err := client.publicAddress(ctx, nodeAddress)
		if err != nil {
			return fmt.Errorf("could not get current serverAddress: %v", err)
		}

		floatingIP, err := client.floatingIP(ctx)
		if err != nil {
			return err
		}

		if serverAddress.ID != floatingIP.Server.ID {
			fmt.Printf("Switching address %s to serverAddress %s.", floatingIP.IP.String(), serverAddress.Name)
			// TODO: Check if FloatingIP.Assign error returns != 200 OK errors
			// I believe you should check the returned response as the returned error only returns if http call fails
			_, _, err := client.HetznerClient.FloatingIP.Assign(ctx, floatingIP, serverAddress)
			if err != nil {
				return fmt.Errorf("could not update floating IP: %v", err)
			}
		} else {
			fmt.Printf("Address %s already assigned to serverAddress %s. Nothing to do.", floatingIP.IP.String(), serverAddress.Name)
		}

		time.Sleep(30 * time.Second)
	}
}

func (client *Client) floatingIP(ctx context.Context) (ip *hcloud.FloatingIP, err error) {
	ips, err := client.HetznerClient.FloatingIP.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch floating IPs: %v", err)
	}

	for _, ip := range ips {
		if ip.IP.Equal(net.ParseIP(client.Configuration.Address)) {
			return ip, nil
		}
	}

	return nil, fmt.Errorf("IP address %s not allocated", client.Configuration.Address)
}

func (client *Client) publicAddress(ctx context.Context, ip net.IP) (server *hcloud.Server, err error) {
	servers, err := client.HetznerClient.Server.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch servers: %v", err)
	}

	for _, server := range servers {
		if server.PublicNet.IPv4.IP.Equal(ip) {
			return server, nil
		}
	}
	return nil, fmt.Errorf("no server with IP address %s found", ip.String())
}

func (client *Client) nodeAddress() (address net.IP, err error) {
	hostname := os.Getenv("HOSTNAME")
	namespace := os.Getenv("NAMESPACE")
	pods, err := client.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	var nodeName string
	for _, pod := range pods.Items {
		if pod.Name == hostname {
			nodeName = pod.Spec.NodeName
			break
		}
	}

	nodes, err := client.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	var addresses []corev1.NodeAddress
	for _, node := range nodes.Items {
		if node.Name == nodeName {
			addresses = node.Status.Addresses
			break
		}
	}

	for _, address := range addresses {
		if address.Type == corev1.NodeExternalIP {
			return net.ParseIP(address.Address), nil
		}
	}
	return nil, fmt.Errorf("could not find address for current node")
}
