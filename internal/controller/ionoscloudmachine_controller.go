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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/ipam"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// IonosCloudMachineReconciler reconciles a IonosCloudMachine object.
type IonosCloudMachineReconciler struct {
	client.Client
	scheme *runtime.Scheme
	locker *locker.Locker
}

// NewIonosCloudMachineReconciler creates a new IonosCloudMachineReconciler.
func NewIonosCloudMachineReconciler(mgr ctrl.Manager) *IonosCloudMachineReconciler {
	r := &IonosCloudMachineReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		locker: locker.New(),
	}
	return r
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/finalizers,verbs=update

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *IonosCloudMachineReconciler) Reconcile(
	ctx context.Context,
	ionosCloudMachine *infrav1.IonosCloudMachine,
) (_ ctrl.Result, retErr error) {
	logger := ctrl.LoggerFrom(ctx)

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, ionosCloudMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("Machine controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

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

	clusterScope, err := r.getClusterScope(ctx, cluster, ionosCloudMachine)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting infra provider cluster or control plane object: %w", err)
	}
	if clusterScope == nil {
		logger.Info("IONOS Cloud machine is not ready yet")
		return ctrl.Result{}, nil
	}

	// Create the machine scope
	machineScope, err := scope.NewMachine(scope.MachineParams{
		Client:       r.Client,
		Machine:      machine,
		ClusterScope: clusterScope,
		IonosMachine: ionosCloudMachine,
		Locker:       r.locker,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}

	defer func() {
		if err := machineScope.Finalize(); err != nil {
			retErr = errors.Join(err, retErr)
		}
	}()

	cloudService, err := createServiceFromCluster(ctx, r.Client, clusterScope.IonosCluster, logger)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "unable to create IONOS Cloud client")
			// Secret is missing, we try again after some time.
			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to create ionos client: %w", err)
	}

	if err != nil {
		return ctrl.Result{}, errors.New("could not create machine service")
	}
	if !ionosCloudMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, cloudService)
	}

	return r.reconcileNormal(ctx, machineScope, cloudService)
}

func (r *IonosCloudMachineReconciler) reconcileNormal(
	ctx context.Context, machineScope *scope.Machine, cloudService *cloud.Service,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(4).Info("Reconciling IonosCloudMachine")

	if machineScope.HasFailed() {
		log.Info("Error state detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	if !r.isInfrastructureReady(ctx, machineScope) {
		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(machineScope.IonosMachine, infrav1.MachineFinalizer) {
		if err := machineScope.PatchObject(); err != nil {
			err = fmt.Errorf("unable to update finalizer on object: %w", err)
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
		log.Error(err, "Error when trying to determine inflight request states")
	}

	if requeue {
		log.Info("Request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	ipamHelper := ipam.NewHelper(r.Client, log)
	reconcileSequence := []serviceReconcileStep[scope.Machine]{
		{"ReconcileLAN", cloudService.ReconcileLAN},
		{"ReconcileIPAddressClaims", ipamHelper.ReconcileIPAddresses},
		{"ReconcileServer", cloudService.ReconcileServer},
		{"ReconcileIPFailover", cloudService.ReconcileIPFailover},
		{"FinalizeMachineProvisioning", cloudService.FinalizeMachineProvisioning},
	}

	for _, step := range reconcileSequence {
		if requeue, err := step.fn(ctx, machineScope); err != nil || requeue {
			if err != nil {
				err = fmt.Errorf("error in step %s: %w", step.name, err)
			}

			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *IonosCloudMachineReconciler) reconcileDelete(
	ctx context.Context, machineScope *scope.Machine, cloudService *cloud.Service,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	requeue, err := r.checkRequestStates(ctx, machineScope, cloudService)
	if err != nil {
		// In case the request state cannot be determined, we want to continue with the
		// reconciliation loop. This is to avoid getting stuck in a state where we cannot
		// proceed with the reconciliation and are stuck in a loop.
		//
		// In any case we log the error.
		log.Error(err, "Error when trying to determine inflight request states")
	}

	if requeue {
		log.Info("Deletion request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	ipamHelper := ipam.NewHelper(r.Client, log)
	reconcileSequence := []serviceReconcileStep[scope.Machine]{
		// NOTE(avorima): NICs, which are configured in an IP failover configuration, cannot be deleted
		// by a request to delete the server. Therefore, during deletion, we need to remove the NIC from
		// the IP failover configuration.
		{"ReconcileIPFailoverDeletion", cloudService.ReconcileIPFailoverDeletion},
		{"ReconcileServerDeletion", cloudService.ReconcileServerDeletion},
		{"ReconcileLANDeletion", cloudService.ReconcileLANDeletion},
		{"ReconcileFailoverIPBlockDeletion", cloudService.ReconcileFailoverIPBlockDeletion},
		{"ReconcileIPAddressClaimsDeletion", ipamHelper.ReconcileIPAddresses},
	}

	for _, step := range reconcileSequence {
		if requeue, err := step.fn(ctx, machineScope); err != nil || requeue {
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
func (*IonosCloudMachineReconciler) checkRequestStates(
	ctx context.Context,
	machineScope *scope.Machine,
	cloudService *cloud.Service,
) (requeue bool, retErr error) {
	log := ctrl.LoggerFrom(ctx)
	// check cluster-wide request
	if req, exists := machineScope.ClusterScope.GetCurrentRequestByDatacenter(machineScope.DatacenterID()); exists {
		status, message, err := cloudService.GetRequestStatus(ctx, req.RequestPath)
		if err != nil {
			retErr = fmt.Errorf("could not get request status: %w", err)
		} else {
			requeue, retErr = withStatus(status, message, &log,
				func() error {
					// remove the request from the status and patch the cluster
					machineScope.ClusterScope.DeleteCurrentRequestByDatacenter(machineScope.DatacenterID())
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
			requeue, _ = withStatus(status, message, &log,
				func() error {
					// no need to patch the machine here as it will be patched
					// after the machine reconciliation is done.
					log.V(4).Info("Request is done, clearing it from the status")
					machineScope.IonosMachine.DeleteCurrentRequest()
					return nil
				},
			)
		}
	}

	return requeue, retErr
}

func (*IonosCloudMachineReconciler) isInfrastructureReady(ctx context.Context, ms *scope.Machine) bool {
	log := ctrl.LoggerFrom(ctx)
	// Make sure the infrastructure is ready.
	if !ms.ClusterScope.Cluster.Status.InfrastructureReady {
		log.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(
			ms.IonosMachine,
			infrav1.MachineProvisionedCondition,
			infrav1.WaitingForClusterInfrastructureReason,
			clusterv1.ConditionSeverityInfo, "")

		return false
	}

	// Make sure to wait until the data secret was created
	if ms.Machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap data secret is not available yet")
		conditions.MarkFalse(
			ms.IonosMachine,
			infrav1.MachineProvisionedCondition,
			infrav1.WaitingForBootstrapDataReason,
			clusterv1.ConditionSeverityInfo, "",
		)

		return false
	}

	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *IonosCloudMachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.IonosCloudMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(
				util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind(infrav1.IonosCloudMachineType)))).
		Complete(reconcile.AsReconciler[*infrav1.IonosCloudMachine](r.Client, r))
}

func (r *IonosCloudMachineReconciler) getClusterScope(
	ctx context.Context, cluster *clusterv1.Cluster, ionosCloudMachine *infrav1.IonosCloudMachine,
) (*scope.Cluster, error) {
	var clusterScope *scope.Cluster
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
	clusterScope, err = scope.NewCluster(scope.ClusterParams{
		Client:       r.Client,
		Cluster:      cluster,
		IonosCluster: ionosCloudCluster,
		Locker:       r.locker,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster scope: %w", err)
	}

	return clusterScope, nil
}
