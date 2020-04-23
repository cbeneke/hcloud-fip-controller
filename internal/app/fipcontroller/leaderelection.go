package fipcontroller

import (
	"context"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

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
	_, err := controller.KubernetesClient.CoordinationV1().Leases(controller.Configuration.Namespace).Get(ctx, controller.Configuration.LeaseName, metav1.GetOptions{})
	if err == nil || !strings.Contains(err.Error(), "the leader is shutting down") {
		controller.Logger.Fatalf("Expected to get an error when trying to make a client call: %v", err)
	}
}

func (controller *Controller) onStartedLeading(ctx context.Context) {
	controller.Logger.Info("Started leading")
	err := controller.Run(ctx)
	if err != nil {
		controller.Logger.Fatalf("Could not run controller: %v", err)
	}
}

func (controller *Controller) onStoppedLeading() {
	controller.Logger.Info("Stopped leading")
}
