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

// Package controller contains the main reconciliation logic for this application.
// the controllers make sure to perform actions according to the state of the resource,
// which is being watched.
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
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// IonosCloudClusterReconciler reconciles a IonosCloudCluster object.
type IonosCloudClusterReconciler struct {
	client.Client
	scheme *runtime.Scheme
	locker *locker.Locker
}

// NewIonosCloudClusterReconciler creates a new IonosCloudClusterReconciler.
func NewIonosCloudClusterReconciler(mgr ctrl.Manager) *IonosCloudClusterReconciler {
	r := &IonosCloudClusterReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		locker: locker.New(),
	}
	return r
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters/finalizers,verbs=update

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *IonosCloudClusterReconciler) Reconcile(
	ctx context.Context,
	ionosCloudCluster *infrav1.IonosCloudCluster,
) (_ ctrl.Result, retErr error) {
	logger := ctrl.LoggerFrom(ctx)

	cluster, err := util.GetOwnerCluster(ctx, r.Client, ionosCloudCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info("Waiting for cluster controller to set OwnerRef on IonosCloudCluster")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	if annotations.IsPaused(cluster, ionosCloudCluster) {
		logger.Info("Either IonosCloudCluster or owner cluster is marked as paused. Reconciliation is skipped")
		return ctrl.Result{}, nil
	}

	clusterScope, err := scope.NewCluster(scope.ClusterParams{
		Client:       r.Client,
		Cluster:      cluster,
		IonosCluster: ionosCloudCluster,
		Locker:       r.locker,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to create scope %w", err)
	}

	// Make sure to persist the changes to the cluster before exiting the function.
	defer func() {
		if err := clusterScope.Finalize(); err != nil {
			retErr = errors.Join(err, retErr)
		}
	}()

	cloudService, err := createServiceFromCluster(ctx, r.Client, ionosCloudCluster, logger)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "unable to create IONOS Cloud client")
			// Secret is missing, we try again after some time.
			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to create ionos client: %w", err)
	}

	if !ionosCloudCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope, cloudService)
	}

	return r.reconcileNormal(ctx, clusterScope, cloudService)
}

func (r *IonosCloudClusterReconciler) reconcileNormal(
	ctx context.Context,
	clusterScope *scope.Cluster,
	cloudService *cloud.Service,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	controllerutil.AddFinalizer(clusterScope.IonosCluster, infrav1.ClusterFinalizer)
	log.V(4).Info("Reconciling IonosCloudCluster")

	requeue, err := r.checkRequestStatus(ctx, clusterScope, cloudService)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error when trying to determine in-flight request states: %w", err)
	}
	if requeue {
		log.Info("Request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	reconcileSequence := []serviceReconcileStep[scope.Cluster]{
		{"ReconcileControlPlaneEndpoint", cloudService.ReconcileControlPlaneEndpoint},
	}
	for _, step := range reconcileSequence {
		if requeue, err := step.fn(ctx, clusterScope); err != nil || requeue {
			if err != nil {
				err = fmt.Errorf("error in step %s: %w", step.name, err)
			}

			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
	}

	conditions.MarkTrue(clusterScope.IonosCluster, infrav1.IonosCloudClusterReady)
	clusterScope.IonosCluster.Status.Ready = true
	return ctrl.Result{}, nil
}

func (r *IonosCloudClusterReconciler) reconcileDelete(
	ctx context.Context, clusterScope *scope.Cluster, cloudService *cloud.Service,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.Cluster.DeletionTimestamp.IsZero() {
		log.Error(errors.New("deletion was requested but owning cluster wasn't deleted"),
			"unable to delete IonosCloudCluster")
		// No need to reconcile again until the owning cluster was deleted.
		return ctrl.Result{}, nil
	}

	requeue, err := r.checkRequestStatus(ctx, clusterScope, cloudService)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error when trying to determine in-flight request states: %w", err)
	}
	if requeue {
		log.Info("Request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	machines, err := clusterScope.ListMachines(ctx, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(machines) > 0 {
		log.Info("Waiting for all IonosCloudMachines to be deleted", "remaining", len(machines))
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	reconcileSequence := []serviceReconcileStep[scope.Cluster]{
		{"ReconcileControlPlaneEndpointDeletion", cloudService.ReconcileControlPlaneEndpointDeletion},
	}
	for _, step := range reconcileSequence {
		if requeue, err := step.fn(ctx, clusterScope); err != nil || requeue {
			if err != nil {
				err = fmt.Errorf("error in step %s: %w", step.name, err)
			}

			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
	}
	if err := removeCredentialsFinalizer(ctx, r.Client, clusterScope.IonosCluster); err != nil {
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(clusterScope.IonosCluster, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

func (*IonosCloudClusterReconciler) checkRequestStatus(
	ctx context.Context, clusterScope *scope.Cluster, cloudService *cloud.Service,
) (requeue bool, retErr error) {
	log := ctrl.LoggerFrom(ctx)
	ionosCluster := clusterScope.IonosCluster
	if req := ionosCluster.Status.CurrentClusterRequest; req != nil {
		status, message, err := cloudService.GetRequestStatus(ctx, req.RequestPath)
		if err != nil {
			retErr = fmt.Errorf("could not get request status: %w", err)
		} else {
			requeue, retErr = withStatus(status, message, &log,
				func() error {
					ionosCluster.DeleteCurrentClusterRequest()
					return nil
				},
			)
		}
	}
	return requeue, retErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *IonosCloudClusterReconciler) SetupWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	options controller.Options,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.IonosCloudCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Watches(&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(
				util.ClusterToInfrastructureMapFunc(
					ctx,
					infrav1.GroupVersion.WithKind(infrav1.IonosCloudClusterKind),
					r.Client, &infrav1.IonosCloudCluster{},
				),
			),
			builder.WithPredicates(predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx))),
		).
		Complete(reconcile.AsReconciler[*infrav1.IonosCloudCluster](r.Client, r))
}
