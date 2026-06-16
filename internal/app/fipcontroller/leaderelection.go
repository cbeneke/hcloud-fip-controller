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
		RenewDeadline:   time.Duration(controller.Configuration.LeaseRenewDeadline) * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: controller.onStartedLeading,
			OnStoppedLeading: controller.onStoppedLeading,
			OnNewLeader:      controller.onNewLeader,
		},
	}
	return
}

// RunWithLeaderElection starts a leaderelection and will run the main logic when it becomes the leader
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

// onNewLeader fires on every participant the first time a leader is observed,
// whether this instance won the election or is following another leader. At
// that point the leader election loop is functioning, so the pod is marked
// ready. Standby pods stay ready as well, so they can take over quickly.
func (controller *Controller) onNewLeader(identity string) {
	controller.Logger.Infof("Observed leader: %s", identity)
	if controller.HealthServer != nil {
		controller.HealthServer.SetReady(true)
	}
}
