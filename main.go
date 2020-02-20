package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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

type SlackRequestBody struct {
	Text string `json:"text"`
}

func slackNotification(podName *v1.Pod, container string, restartCount int32) {
	webhookURL := "https://hooks.slack.com/services/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"

	attachment1 := slack.Attachment{}
	attachment1.AddField(slack.Field{Title: "Pod Name", Value: podName.Name}).AddField(slack.Field{Title: "Container Name", Value: container})
	attachment1.AddField(slack.Field{Title: "Restarted", Value: "true"})
	attachment1.AddAction(slack.Action{Type: "button", Text: "Open Jira 🛫", Url: "", Style: "primary"})
	attachment1.AddAction(slack.Action{Type: "button", Text: "Cancel", Url: "", Style: "danger"})
	payload := slack.Payload{
		Text:        "Pod Crash Notification Alert",
		Username:    "Kube Bot",
		Channel:     "#kubernetes-demo",
		IconEmoji:   ":monkey_face:",
		Attachments: []slack.Attachment{attachment1},
	}
	err := slack.Send(webhookURL, "", payload)
	if len(err) > 0 {
		fmt.Printf("error: %s\n", err)
	}
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
			log.Info("Reconciling container: " + container)
			slackNotification(pod, container, restartCount)
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
