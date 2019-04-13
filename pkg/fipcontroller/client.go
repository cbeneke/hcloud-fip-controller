package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net"
	"os"
	"time"
)

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

func Run(ctx context.Context, client *Client) error {
	for {
		serverAddress, err := GetKubeNodeAddress(client)
		if err != nil {
			return fmt.Errorf("could not get kubernetes server address: %v", err)
		}

		server, err := GetServerByPublicAddress(ctx, client, serverAddress)
		if err != nil {
			return fmt.Errorf("could not get current server: %v", err)
		}

		floatingIP, err := GetFipFromClient(ctx, client)

		if server.ID != floatingIP.Server.ID {
			fmt.Printf("Switching address %s to server %s.", floatingIP.IP.String(), server.Name)
			_, _, err := client.HetznerClient.FloatingIP.Assign(ctx, floatingIP, server)
			if err != nil {
				return fmt.Errorf("could not update floating IP: %v", err)
			}
		} else {
			fmt.Printf("Address %s already assigned to server %s. Nothing to do.", floatingIP.IP.String(), server.Name)
		}

		time.Sleep(30 * time.Second)
	}
}

func GetFipFromClient(ctx context.Context, client *Client) (ip *hcloud.FloatingIP, err error) {
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

func GetServerByPublicAddress(ctx context.Context, client *Client, ip net.IP) (server *hcloud.Server, err error) {
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

func GetKubeNodeAddress(client *Client) (address net.IP, err error) {
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
