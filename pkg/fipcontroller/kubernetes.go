package fipcontroller

import (
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	for _, address := range addresses {
		// TODO: Make address type configurable
		if address.Type == corev1.NodeInternalIP {
			return net.ParseIP(address.Address), nil
		}
	}
	return nil, fmt.Errorf("could not find address for current node")
}
