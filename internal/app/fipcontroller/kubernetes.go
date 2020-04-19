package fipcontroller

import (
	"fmt"
	"net"

	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"

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

/*
 * Search and return the IP address of a given kubernetes node name.
 *  Will return first found internal or external IP depending on nodeAddressType parameter
 */
func (controller *Controller) nodeAddressList(nodeAddressType configuration.NodeAddressType) (addressList []net.IP, err error) {
	nodes, err := controller.KubernetesClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list nodes: %v", err)
	}
	controller.Logger.Debugf("Found %d nodes", len(nodes.Items))

	for _, node := range nodes.Items {
		// Skip unhealthy nodes
		if !isNodeHealthy(node) {
			continue
		}

		addresses := node.Status.Addresses
		controller.Logger.Debugf("Found %d addresses for node %s", len(addresses), node.Name)

		checkAddressType := corev1.NodeExternalIP
		if nodeAddressType == configuration.NodeAddressTypeInternal {
			checkAddressType = corev1.NodeInternalIP
		}
		controller.Logger.Debugf("Using address type '%s' for node %s", checkAddressType, node.Name)

		for _, address := range addresses {
			if address.Type == checkAddressType {
				addressList = append(addressList, net.ParseIP(address.Address))
				break
			}
		}
		// TODO: check nothing found
	}
	return
}

/*
 * Check if node is healthy
 */
func isNodeHealthy(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
