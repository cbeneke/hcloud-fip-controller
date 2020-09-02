package fipcontroller

import (
	"context"
	"fmt"
	"k8s.io/client-go/util/retry"
	"net"
	"strings"

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

// Search and return the IP address of a given kubernetes node name.
// Will return first found internal or external IP depending on nodeAddressType parameter
func (controller *Controller) nodeAddressList(ctx context.Context, nodeAddressType configuration.NodeAddressType) (addressList []net.IP, err error) {
	podLabelSelector, err := controller.createPodLabelSelector(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get information about pod: %v", err)
	}

	// Try to get deployment pods if certain label is specified
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = podLabelSelector
	var pods *corev1.PodList

	err = retry.OnError(controller.Backoff, alwaysRetry, func() error {
		pods, err = controller.KubernetesClient.CoreV1().Pods(controller.Configuration.Namespace).List(ctx, listOptions)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not list nodes: %v", err)
	}
	controller.Logger.Debugf("Found %d pods", len(pods.Items))

	for _, pod := range pods.Items {
		address := net.ParseIP(pod.Status.HostIP)
		addressList = append(addressList, address)
	}
	if len(addressList) > 0 {
		controller.Logger.Debugf("Found %d ips from pods", len(addressList))
		return
	}

	// Create list options with optional labelSelector
	listOptions = metav1.ListOptions{}
	if controller.Configuration.NodeLabelSelector != "" {
		listOptions.LabelSelector = controller.Configuration.NodeLabelSelector
	}
	var nodes *corev1.NodeList

	err = retry.OnError(controller.Backoff, alwaysRetry, func() error {
		nodes, err = controller.KubernetesClient.CoreV1().Nodes().List(ctx, listOptions)
		return err
	})
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

		address := searchForAddress(addresses, checkAddressType)
		if address == nil {
			return nil, fmt.Errorf("coud not find address for node %s", node.Name)
		}
		addressList = append(addressList, address)
	}

	if len(addressList) < 1 {
		return nil, fmt.Errorf("could not find any healthy nodes")
	}

	return
}

// Check if node is healthy
func isNodeHealthy(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func searchForAddress(addresses []corev1.NodeAddress, checkAddressType corev1.NodeAddressType) net.IP {
	for _, address := range addresses {
		if address.Type == checkAddressType {
			return net.ParseIP(address.Address)
		}
	}
	return nil
}

func (controller *Controller) createPodLabelSelector(ctx context.Context) (string, error) {
	if controller.Configuration.PodLabelSelector != "" {
		return controller.Configuration.PodLabelSelector, nil
	}

	if controller.Configuration.PodName == "" {
		controller.Logger.Warn("no pod name specified in configuration, all pods in namespace will be used")
		return "", nil
	}

	var pod *corev1.Pod
	var err error
	err = retry.OnError(controller.Backoff, alwaysRetry, func() error {
		pod, err = controller.KubernetesClient.CoreV1().Pods(controller.Configuration.Namespace).Get(ctx, controller.Configuration.PodName, metav1.GetOptions{})
		return err
	})
	if err != nil {
		return "", fmt.Errorf("Could not get pod information: %v", err)
	}
	if len(pod.Labels) < 1 {
		controller.Logger.Warnf("fip-controller pod has no labels, all pods in namespace will be used")
		return "", nil
	}

	var stringBuilder strings.Builder
	for key, value := range pod.Labels {
		fmt.Fprintf(&stringBuilder, "%s=%s,", key, value)
	}
	labelSelector := stringBuilder.String()
	labelSelector = labelSelector[:stringBuilder.Len()-1] // remove trailing ,
	controller.Logger.Debugf("pod label selector created: %s", labelSelector)
	return labelSelector, nil
}
