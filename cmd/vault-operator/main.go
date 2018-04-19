// Copyright 2018 The vault-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/coreos/vault-operator/pkg/operator"
	"github.com/coreos/vault-operator/pkg/util/k8sutil"
	"github.com/coreos/vault-operator/pkg/util/probe"
	"github.com/coreos/vault-operator/version"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

func main() {
	namespace := os.Getenv("MY_POD_NAMESPACE")
	if len(namespace) == 0 {
		logrus.Fatalf("must set env MY_POD_NAMESPACE")
	}
	name := os.Getenv("MY_POD_NAME")
	if len(name) == 0 {
		logrus.Fatalf("must set env MY_POD_NAME")
	}

	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("vault-operator Version: %v", version.Version)
	logrus.Infof("Git SHA: %s", version.GitSHA)

	kubecfg, err := k8sutil.InClusterConfig()
	if err != nil {
		panic(err)
	}
	kubecli := kubernetes.NewForConfigOrDie(kubecfg)

	http.HandleFunc(probe.HTTPReadyzEndpoint, probe.ReadyzHandler)
	go http.ListenAndServe("0.0.0.0:8080", nil)

	id, err := os.Hostname()
	if err != nil {
		logrus.Fatalf("failed to get hostname: %v", err)
	}
	rl, err := resourcelock.New(resourcelock.EndpointsResourceLock,
		namespace,
		"vault-operator",
		kubecli.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: createRecorder(kubecli, name, namespace),
		})
	if err != nil {
		logrus.Fatalf("error creating lock: %v", err)
	}

	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				logrus.Fatalf("leader election lost")
			},
		},
	})
	// unreachable
}

func run(stop <-chan struct{}) {
	v := operator.New()
	err := v.Start(context.TODO())
	if err != nil {
		// If we don't exit the program,
		// there is another go routine that keeps renewing the LE lock.
		logrus.Fatalf("operator stopped with %v", err)
	}
}

func createRecorder(kubecli kubernetes.Interface, name, namespace string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubecli.Core().RESTClient()).Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: name})
}
