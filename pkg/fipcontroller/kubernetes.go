package fipcontroller

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func kubernetesClient() (*kubernetes.Clientset, error) {
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

	var addresses []corev1.NodeAddress
	for _, node := range nodes.Items {
		if node.Name == nodeName {
			addresses = node.Status.Addresses
			break
		}
	}

	checkAddressType := corev1.NodeExternalIP
	if nodeAddressType == "internal" {
		checkAddressType = corev1.NodeInternalIP
	}

	for _, address := range addresses {
		if address.Type == checkAddressType {
			return net.ParseIP(address.Address), nil
		}
	}
	return nil, fmt.Errorf("could not find address for node %s", nodeName)
}

func (controller *Controller) leaseLock(id string) (lock *resourcelock.LeaseLock) {
	lock = &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      controller.Configuration.LeaseName,
			Namespace: controller.Configuration.Namespace,
		},
		Client: controller.KubernetesClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}
	return
}

func (controller *Controller) leaderElectionConfig() (config leaderelection.LeaderElectionConfig) {
	config = leaderelection.LeaderElectionConfig{
		Lock:            controller.leaseLock(controller.Configuration.PodName),
		ReleaseOnCancel: true,
		LeaseDuration:   time.Duration(controller.Configuration.LeaseDuration) * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: controller.onStartedLeading,
			OnStoppedLeading: controller.onStoppedLeading,
		},
	}
	return
}

func (controller *Controller) RunWithLeaderElection(ctx context.Context) {
	leaderelection.RunOrDie(ctx, controller.leaderElectionConfig())

	// because the context is closed, the client should report errors
	_, err := controller.KubernetesClient.CoordinationV1().Leases(controller.Configuration.Namespace).Get(controller.Configuration.LeaseName, metav1.GetOptions{})
	if err == nil || !strings.Contains(err.Error(), "the leader is shutting down") {
		log.Fatalf("%s: expected to get an error when trying to make a client call: %v", controller.Configuration.PodName, err)
	}
}

func (controller *Controller) onStartedLeading(ctx context.Context) {
	log.Println("Became Leader")
	err := controller.Run(ctx)
	if err != nil {
		log.Fatalf("%s: could not run controller: %v\n", controller.Configuration.PodName, err)
	}
}

func (controller *Controller) onStoppedLeading() {
	log.Println("Stopped leading")
}
