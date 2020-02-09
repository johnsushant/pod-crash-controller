package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var restartList map[string]int32

func slackNotification(message string) {
	println(message)
}

type ReconcilePod struct {
	Client client.Client
	Logger logr.Logger
}

func (r *ReconcilePod) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.Logger
	pod := &corev1.Pod{}
	err := r.Client.Get(context.Background(), request.NamespacedName, pod)
	if errors.IsNotFound(err) {
		log.Error(nil, "Pod Not Found. Could have been deleted")
		return reconcile.Result{}, nil
	}
	if err != nil {
		log.Error(err, "Error fetching pod. Going to requeue")
		return reconcile.Result{Requeue: true}, err
	}
	for i := range pod.Status.ContainerStatuses {
		container := pod.Status.ContainerStatuses[i].Name
		restartCount := pod.Status.ContainerStatuses[i].RestartCount
		identifier := pod.Name + pod.Status.ContainerStatuses[i].Name
		if _, ok := restartList[identifier]; !ok {
			restartList[identifier] = restartCount
		} else if restartList[identifier] < restartCount {
			slackNotification(fmt.Sprintf("%s container of %s pod restarted. Total restart count: %d", container, pod.Name, restartCount))
			restartList[identifier] = restartCount
		}
	}
	return reconcile.Result{}, nil
}

func main() {
	log := zapr.NewLogger(zap.NewExample()).WithName("pod-crash-controller")
	restartList = make(map[string]int32)

	log.Info("Setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		log.Error(err, "Unable to setup manager. Please check if KUBECONFIG is available")
		os.Exit(1)
	}

	log.Info("Setting up controller")
	ctrl, err := controller.New("pod-crash-controller", mgr, controller.Options{
		Reconciler: &ReconcilePod{Client: mgr.GetClient(), Logger: log},
	})
	if err != nil {
		log.Error(err, "Failed to setup controller")
		os.Exit(1)
	}

	log.Info("Watching Pods")
	if err := ctrl.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}); err != nil {
		log.Error(err, "Failed to watch pods")
		os.Exit(1)
	}

	log.Info("Starting the manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Failed to start manager")
		os.Exit(1)
	}
}
