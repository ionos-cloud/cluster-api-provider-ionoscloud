/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main performs initialization and starts the manager
// which will continuously reconcile on application defined resources.
package main

import (
	"flag"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/flags"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	iccontroller "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/controller"
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	healthProbeAddr      string
	enableLeaderElection bool
	diagnosticOptions    = flags.DiagnosticsOptions{}

	icClusterConcurrency int
	icMachineConcurrency int
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(infrav1.AddToScheme(scheme))
	utilruntime.Must(ipamv1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

// Add RBAC for the authorized diagnostics endpoint.
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create

func main() {
	ctrl.SetLogger(klog.Background())
	initFlags()
	pflag.Parse()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                flags.GetDiagnosticsOptions(diagnosticOptions),
		HealthProbeBindAddress: healthProbeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "15f3d3ca.cluster.x-k8s.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	if err = iccontroller.NewIonosCloudClusterReconciler(mgr).SetupWithManager(
		ctx,
		mgr,
		controller.Options{MaxConcurrentReconciles: icClusterConcurrency},
	); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IonosCloudCluster")
		os.Exit(1)
	}
	if err = iccontroller.NewIonosCloudMachineReconciler(mgr).SetupWithManager(
		mgr,
		controller.Options{MaxConcurrentReconciles: icMachineConcurrency},
	); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IonosCloudMachine")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// initFlags parses the command line flags.
func initFlags() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	flags.AddDiagnosticsOptions(pflag.CommandLine, &diagnosticOptions)
	pflag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.IntVar(&icClusterConcurrency, "ionoscloudcluster-concurrency", 1,
		"Number of IonosCloudClusters to process simultaneously")
	pflag.IntVar(&icMachineConcurrency, "ionoscloudmachine-concurrency", 1,
		"Number of IonosCloudMachines to process simultaneously")
}
