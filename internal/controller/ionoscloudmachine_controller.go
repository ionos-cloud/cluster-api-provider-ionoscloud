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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
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

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IonosCloudMachine object against the actual cluster state, and then
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
		logger.Info("Machine controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", klog.KObj(machine))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	if annotations.IsPaused(cluster, ionosCloudMachine) {
		logger.Info("IONOS Cloud machine or linked cluster is marked as paused, not reconciling")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))

	clusterScope, err := r.getClusterScope(ctx, cluster, ionosCloudMachine)
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
		return ctrl.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}

	defer func() {
		if err := machineScope.Finalize(); err != nil {
			retErr = errors.Join(err, retErr)
		}
	}()

	cloudService, err := cloud.NewService(ctx, machineScope, r.IonosCloudClient)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not create machine service")
	}
	if !ionosCloudMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, cloudService)
	}

	return r.reconcileNormal(ctx, machineScope, cloudService)
}

func (r *IonosCloudMachineReconciler) reconcileNormal(
	ctx context.Context,
	machineScope *scope.MachineScope,
	cloudService *cloud.Service,
) (ctrl.Result, error) {
	machineScope.V(4).Info("Reconciling IonosCloudMachine")

	if machineScope.HasFailed() {
		machineScope.Info("Error state detected, skipping reconciliation")
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

	requeue, err := r.checkRequestStates(ctx, machineScope, cloudService)
	if err != nil {
		// In case the request state cannot be determined, we want to continue with the
		// reconciliation loop. This is to avoid getting stuck in a state where we cannot
		// proceed with the reconciliation and are stuck in a loop.
		//
		// In any case we log the error.
		machineScope.Error(err, "Error when trying to determine inflight request states")
	}

	if requeue {
		machineScope.Info("Request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	// TODO(piepmatz): This is not thread-safe, but needs to be. Add locking.
	reconcileSequence := []serviceReconcileStep{
		{name: "ReconcileLAN", reconcileFunc: cloudService.ReconcileLAN},
		{name: "ReconcileServer", reconcileFunc: cloudService.ReconcileServer},
		{name: "ReconcileIPFailover", reconcileFunc: cloudService.ReconcileIPFailover},
		{name: "FinalizeMachineProvisioning", reconcileFunc: cloudService.FinalizeMachineProvisioning},
	}

	for _, step := range reconcileSequence {
		if requeue, err := step.reconcileFunc(); err != nil || requeue {
			if err != nil {
				err = fmt.Errorf("error in step %s: %w", step.name, err)
			}

			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *IonosCloudMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, cloudService *cloud.Service) (ctrl.Result, error) {
	// TODO(piepmatz): This is not thread-safe, but needs to be. Add locking.
	//  Moreover, should only be attempted if it's the last machine using that LAN. We should check that our machines
	//  at least, but need to accept that users added their own infrastructure into our LAN (in that case a LAN deletion
	//  attempt will be denied with HTTP 422).
	requeue, err := r.checkRequestStates(ctx, machineScope, cloudService)
	if err != nil {
		// In case the request state cannot be determined, we want to continue with the
		// reconciliation loop. This is to avoid getting stuck in a state where we cannot
		// proceed with the reconciliation and are stuck in a loop.
		//
		// In any case we log the error.
		machineScope.Error(err, "Error when trying to determine inflight request states")
	}

	if requeue {
		machineScope.Info("Deletion request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	reconcileSequence := []serviceReconcileStep{
		// NOTE(avorima): NICs, which are configured in an IP failover configuration, cannot be deleted
		// by a request to delete the server. Therefore, during deletion, we need to remove the NIC from
		// the IP failover configuration.
		{name: "ReconcileIPFailoverDeletion", reconcileFunc: cloudService.ReconcileIPFailoverDeletion},
		{name: "ReconcileServerDeletion", reconcileFunc: cloudService.ReconcileServerDeletion},
		{name: "ReconcileLANDeletion", reconcileFunc: cloudService.ReconcileLANDeletion},
	}

	for _, step := range reconcileSequence {
		if requeue, err := step.reconcileFunc(); err != nil || requeue {
			if err != nil {
				err = fmt.Errorf("error in step %s: %w", step.name, err)
			}

			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
	}

	controllerutil.RemoveFinalizer(machineScope.IonosMachine, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil
}

// Before starting with the reconciliation loop,
// we want to check if there is any pending request in the IONOS cluster or machine spec.
// If there is any pending request, we need to check the status of the request and act accordingly.
// Status:
//   - Queued, Running => Requeue the current request
//   - Failed => Log the error and continue also apply the same logic as in Done.
//   - Done => Clear request from the status and continue reconciliation.
func (r *IonosCloudMachineReconciler) checkRequestStates(
	ctx context.Context,
	machineScope *scope.MachineScope,
	cloudService *cloud.Service,
) (requeue bool, retErr error) {
	// check cluster wide request
	ionosCluster := machineScope.ClusterScope.IonosCluster
	if req, exists := ionosCluster.Status.CurrentRequestByDatacenter[machineScope.DatacenterID()]; exists {
		status, message, err := cloudService.GetRequestStatus(ctx, req.RequestPath)
		if err != nil {
			retErr = fmt.Errorf("could not get request status: %w", err)
		} else {
			requeue, retErr = withStatus(status, message, machineScope.Logger,
				func() error {
					// remove the request from the status and patch the cluster
					ionosCluster.DeleteCurrentRequestByDatacenter(machineScope.DatacenterID())
					return machineScope.ClusterScope.PatchObject()
				},
			)
		}
	}

	// check machine related request
	if req := machineScope.IonosMachine.Status.CurrentRequest; req != nil {
		status, message, err := cloudService.GetRequestStatus(ctx, req.RequestPath)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("could not get request status: %w", err))
		} else {
			requeue, _ = withStatus(status, message, machineScope.Logger,
				func() error {
					// no need to patch the machine here as it will be patched
					// after the machine reconciliation is done.
					machineScope.V(4).Info("Request is done, clearing it from the status")
					machineScope.IonosMachine.Status.CurrentRequest = nil
					return nil
				},
			)
		}
	}

	return requeue, retErr
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
	ctx context.Context, cluster *clusterv1.Cluster, ionosCloudMachine *infrav1.IonosCloudMachine,
) (*scope.ClusterScope, error) {
	var clusterScope *scope.ClusterScope
	var err error

	ionosCloudCluster := &infrav1.IonosCloudCluster{}

	infraClusterName := client.ObjectKey{
		Namespace: ionosCloudMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, infraClusterName, ionosCloudCluster); err != nil {
		if apierrors.IsNotFound(err) {
			// Cluster has not yet been created
			return nil, nil
		}
		// We at most expect that the cluster cannot be found.
		// If the error is different, we should return that particular error.
		return nil, err
	}

	// Create the cluster scope
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Cluster:      cluster,
		IonosCluster: ionosCloudCluster,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster scope: %w", err)
	}

	return clusterScope, nil
}
