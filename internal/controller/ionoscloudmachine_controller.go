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

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"

	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// IonosCloudMachineReconciler reconciles a IonosCloudMachine object.
type IonosCloudMachineReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	IonosCloudClient ionoscloud.Client
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IonosMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *IonosCloudMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, retErr error) {
	logger := ctrl.LoggerFrom(ctx)

	ionosCloudMachine := &infrav1.IonosCloudMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, ionosCloudMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, ionosCloudMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("machine controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", klog.KObj(machine))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	if annotations.IsPaused(cluster, ionosCloudMachine) {
		logger.Info("IONOS Cloud machine or linked cluster is marked as paused, not reconciling")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))

	clusterScope, err := r.getClusterScope(ctx, &logger, cluster, ionosCloudMachine)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting infra provider cluster or control plane object: %w", err)
	}
	if clusterScope == nil {
		logger.Info("IONOS Cloud machine is not ready yet")
		return ctrl.Result{}, nil
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:       r.Client,
		Cluster:      cluster,
		Machine:      machine,
		ClusterScope: clusterScope,
		IonosMachine: ionosCloudMachine,
		Logger:       &logger,
	})
	if err != nil {
		logger.Error(err, "failed to create scope")
		return ctrl.Result{}, err
	}

	defer func() {
		if err := machineScope.Finalize(); err != nil {
			retErr = errors.Join(err, retErr)
		}
	}()

	cloudService, err := cloud.NewService(ctx, machineScope)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not create machine service")
	}
	if !ionosCloudMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(cloudService)
	}

	return r.reconcileNormal(machineScope, clusterScope, cloudService)
}

func (r *IonosCloudMachineReconciler) isInfrastructureReady(machineScope *scope.MachineScope) bool {
	// Make sure the infrastructure is ready.
	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(
			machineScope.IonosMachine,
			infrav1.MachineProvisionedCondition,
			infrav1.WaitingForClusterInfrastructureReason,
			clusterv1.ConditionSeverityInfo, "")

		return false
	}

	// Make sure to wait until the data secret was created
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret is not available yet")
		conditions.MarkFalse(
			machineScope.IonosMachine,
			infrav1.MachineProvisionedCondition,
			infrav1.WaitingForBootstrapDataReason,
			clusterv1.ConditionSeverityInfo, "",
		)

		return false
	}

	return true
}

func (r *IonosCloudMachineReconciler) reconcileNormal(machineScope *scope.MachineScope, _ *scope.ClusterScope, cloudService *cloud.Service) (ctrl.Result, error) {
	machineScope.V(4).Info("reconciling IonosCloudMachine")

	if machineScope.HasFailed() {
		machineScope.Info("error state detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	if !r.isInfrastructureReady(machineScope) {
		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(machineScope.IonosMachine, infrav1.MachineFinalizer) {
		if err := machineScope.PatchObject(); err != nil {
			machineScope.Error(err, "unable to update finalizer on object")
			return ctrl.Result{}, err
		}
	}

	// TODO(lubedacht) Check before starting reconciliation if there is any pending request in the Ionos cluster or machine spec
	// 	If there is, query for the request and check the status
	// 	Status:
	//		* Done = Clear request from the status and continue reconciliation
	//		* Queued, Running => Requeue the current request
	// 		* Failed => We need to discuss this, log error and continue (retry last request in the corresponding reconcile function)

	// Ensure that a LAN is created in the datacenter
	if requeue, err := cloudService.ReconcileLAN(); err != nil || requeue {
		if requeue {
			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
		return ctrl.Result{}, fmt.Errorf("could not reconcile LAN %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *IonosCloudMachineReconciler) reconcileDelete(cloudService *cloud.Service) (ctrl.Result, error) {
	requeue, err := cloudService.ReconcileLANDeletion()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not reconcile LAN deletion: %w", err)
	}

	var after time.Duration
	if requeue {
		after = defaultReconcileDuration
	}
	return ctrl.Result{
		RequeueAfter: after,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IonosCloudMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.IonosCloudMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind(infrav1.IonosCloudMachineType)))).
		Complete(r)
}

func (r *IonosCloudMachineReconciler) getClusterScope(
	ctx context.Context, logger *logr.Logger, cluster *clusterv1.Cluster, ionosCloudMachine *infrav1.IonosCloudMachine,
) (*scope.ClusterScope, error) {
	var clusterScope *scope.ClusterScope
	var err error

	ionosCloudCluster := &infrav1.IonosCloudCluster{}

	infraClusterName := client.ObjectKey{
		Namespace: ionosCloudMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, infraClusterName, ionosCloudCluster); err != nil {
		// IonosCloudCluster is not ready
		return nil, nil //nolint:nilerr
	}

	// Create the cluster scope
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       logger,
		Cluster:      cluster,
		IonosCluster: ionosCloudCluster,
		IonosClient:  r.IonosCloudClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to creat cluster scope: %w", err)
	}

	return clusterScope, nil
}
