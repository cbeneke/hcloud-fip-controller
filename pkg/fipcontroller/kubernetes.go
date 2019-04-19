package fipcontroller

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func kubernetesClient() (*kubernetes.Clientset, error) {
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get kubeconfig: %v", err)
	}

	kubernetesClient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not get kubernetes client: %v", err)
	}

	return kubernetesClient, nil
}

func (controller *Controller) nodeAddress() (address net.IP, err error) {
	nodes, err := controller.KubernetesClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list nodes: %v", err)
	}

	var addresses []corev1.NodeAddress
	for _, node := range nodes.Items {
		if node.Name == controller.NodeName {
			addresses = node.Status.Addresses
			break
		}
	}

	checkAddressType := corev1.NodeExternalIP
	if controller.Configuration.NodeAddressType == "internal" {
		checkAddressType = corev1.NodeInternalIP
	}

	for _, address := range addresses {
		// TODO: Make address type configurable
		if address.Type == checkAddressType {
			return net.ParseIP(address.Address), nil
		}
	}
	return nil, fmt.Errorf("could not find address for current node")
}
