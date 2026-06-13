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
	"context"
	"flag"
	"os"

	"github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/cluster-api/controllers/crdmigrator"
	"sigs.k8s.io/cluster-api/util/flags"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	iccontroller "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/controller"
)

const errMsgUnableToCreateController = "unable to create controller"

var (
	scheme                 = runtime.NewScheme()
	setupLog               = ctrl.Log.WithName("setup")
	healthProbeAddr        string
	enableLeaderElection   bool
	managerOptions         = flags.ManagerOptions{}
	skipCRDMigrationPhases []string

	icClusterConcurrency int
	icMachineConcurrency int
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(infrav1.AddToScheme(scheme))
	utilruntime.Must(ipamv1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

// Add RBAC for the authorized diagnostics endpoint.
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create

// Add RBAC for CRDMigrator controller.
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions;customresourcedefinitions/status,verbs=update;patch,resourceNames=ionoscloudclusters.infrastructure.cluster.x-k8s.io;ionoscloudclustertemplates.infrastructure.cluster.x-k8s.io;ionoscloudmachines.infrastructure.cluster.x-k8s.io;ionoscloudmachinetemplates.infrastructure.cluster.x-k8s.io

// Add RBAC for template CRs (needed by CRDMigrator for storage version migration).
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclustertemplates;ionoscloudmachinetemplates,verbs=get;list;watch;patch;update

func main() {
	ctrl.SetLogger(klog.Background())
	initFlags()
	pflag.Parse()

	_, metricsOptions, err := flags.GetManagerOptions(managerOptions)
	if err != nil {
		setupLog.Error(err, "unable to get manager options")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                *metricsOptions,
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
		setupLog.Error(err, errMsgUnableToCreateController, "controller", "IonosCloudCluster")
		os.Exit(1)
	}
	if err = iccontroller.NewIonosCloudMachineReconciler(mgr).SetupWithManager(
		mgr,
		controller.Options{MaxConcurrentReconciles: icMachineConcurrency},
	); err != nil {
		setupLog.Error(err, errMsgUnableToCreateController, "controller", "IonosCloudMachine")
		os.Exit(1)
	}
	if err := setupCRDMigrator(ctx, mgr); err != nil {
		setupLog.Error(err, errMsgUnableToCreateController, "controller", "CRDMigrator")
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

func setupCRDMigrator(ctx context.Context, mgr ctrl.Manager) error {
	crdMigratorConfig := map[client.Object]crdmigrator.ByObjectConfig{
		&infrav1.IonosCloudCluster{}:         {UseCache: true, UseStatusForStorageVersionMigration: true},
		&infrav1.IonosCloudMachine{}:         {UseCache: true, UseStatusForStorageVersionMigration: true},
		&infrav1.IonosCloudClusterTemplate{}: {UseCache: false},
		&infrav1.IonosCloudMachineTemplate{}: {UseCache: false},
	}
	crdMigratorSkipPhases := make([]crdmigrator.Phase, 0, len(skipCRDMigrationPhases))
	for _, p := range skipCRDMigrationPhases {
		crdMigratorSkipPhases = append(crdMigratorSkipPhases, crdmigrator.Phase(p))
	}
	return (&crdmigrator.CRDMigrator{
		Client:                 mgr.GetClient(),
		APIReader:              mgr.GetAPIReader(),
		SkipCRDMigrationPhases: crdMigratorSkipPhases,
		Config:                 crdMigratorConfig,
		// Run with concurrency 1 to avoid overwhelming the apiserver.
	}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: 1})
}

// initFlags parses the command line flags.
func initFlags() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	flags.AddManagerOptions(pflag.CommandLine, &managerOptions)
	pflag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.StringArrayVar(&skipCRDMigrationPhases, "skip-crd-migration-phases", []string{},
		"CRD migration phases to skip. Use when the CAPI CRDMigrator handles migration.")
	pflag.IntVar(&icClusterConcurrency, "ionoscloudcluster-concurrency", 1,
		"Number of IonosCloudClusters to process simultaneously")
	pflag.IntVar(&icMachineConcurrency, "ionoscloudmachine-concurrency", 1,
		"Number of IonosCloudMachines to process simultaneously")
}
