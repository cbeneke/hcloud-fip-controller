package fipcontroller

import (
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func newKubernetesClient() (*kubernetes.Clientset, error) {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get kubeconfig: %v", err)
	}

	kubernetesClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("could not get kubernetes client: %v", err)
	}

	return kubernetesClient, nil
}

func (controller *Controller) nodeAddress(nodeName, nodeAddressType string) (address net.IP, err error) {
	nodes, err := controller.KubernetesClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list nodes: %v", err)
	}
	controller.Logger.Debugf("Found %d nodes", len(nodes.Items))

	var addresses []corev1.NodeAddress
	for _, node := range nodes.Items {
		if node.Name == nodeName {
			addresses = node.Status.Addresses
			break
		}
	}
	controller.Logger.Debugf("Found %d addresses", len(addresses))

	checkAddressType := corev1.NodeExternalIP
	if nodeAddressType == "internal" {
		checkAddressType = corev1.NodeInternalIP
	}
	controller.Logger.Debugf("Using address type %s", checkAddressType)

	for _, address := range addresses {
		if address.Type == checkAddressType {
			return net.ParseIP(address.Address), nil
		}
	}
	return nil, fmt.Errorf("could not find address for node %s", nodeName)
}
