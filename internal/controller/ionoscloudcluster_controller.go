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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// IonosCloudClusterReconciler reconciles a IonosCloudCluster object.
type IonosCloudClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	IonosCloudClient ionoscloud.Client
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters/finalizers,verbs=update

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *IonosCloudClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, retErr error) {
	logger := ctrl.LoggerFrom(ctx)

	// TODO(user): your logic here
	ionosCloudCluster := &infrav1.IonosCloudCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, ionosCloudCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

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

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Cluster:      cluster,
		IonosCluster: ionosCloudCluster,
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

	cloudService, err := cloud.NewService(nil, r.IonosCloudClient)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create cloud service: %w", err)
	}

	if !ionosCloudCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope, cloudService)
	}

	return r.reconcileNormal(ctx, clusterScope, cloudService)
}

func (r *IonosCloudClusterReconciler) reconcileNormal(ctx context.Context, clusterScope *scope.ClusterScope, cloudService *cloud.Service) (ctrl.Result, error) {
	controllerutil.AddFinalizer(clusterScope.IonosCluster, infrav1.ClusterFinalizer)
	conditions.MarkTrue(clusterScope.IonosCluster, infrav1.IonosCloudClusterReady)
	log := ctrl.LoggerFrom(ctx)

	clusterScope.IonosCluster.Status.Ready = true
	log.V(4).Info("Reconciling IonosCloudCluster")

	requeue, err := r.checkRequestStatus(ctx, clusterScope, cloudService)
	if err != nil {
		// In case the request state cannot be determined, we want to continue with the
		// reconciliation loop. This is to avoid getting stuck in a state where we cannot
		// proceed with the reconciliation and are stuck in a loop.
		//
		// In any case we log the error.
		log.Error(err, "Error when trying to determine in-flight request states")
	}
	if requeue {
		log.Info("Request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	reconcileSequence := []serviceReconcileStep{
		{
			"ReconcileControlPlaneEndpoint",
			func() (bool, error) {
				return cloudService.ReconcileControlPlaneEndpoint(ctx, clusterScope)
			},
		},
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

func (r *IonosCloudClusterReconciler) reconcileDelete(
	ctx context.Context, clusterScope *scope.ClusterScope, cloudService *cloud.Service,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if clusterScope.Cluster.DeletionTimestamp.IsZero() {
		log.Error(errors.New("deletion was requested but owning cluster wasn't deleted"), "unable to delete IonosCloudCluster")
		// No need to reconcile again until the owning cluster was deleted.
		return ctrl.Result{}, nil
	}

	requeue, err := r.checkRequestStatus(ctx, clusterScope, cloudService)
	if err != nil {
		// In case the request state cannot be determined, we want to continue with the
		// reconciliation loop. This is to avoid getting stuck in a state where we cannot
		// proceed with the reconciliation and are stuck in a loop.
		//
		// In any case we log the error.
		log.Error(err, "Error when trying to determine in-flight request states")
	}
	if requeue {
		log.Info("Request is still in progress")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	// TODO(lubedacht): check if there are any more machine CRs existing.
	// If there are requeue with an offset.

	reconcileSequence := []serviceReconcileStep{
		{
			"ReconcileControlPlaneEndpointDeletion",
			func() (bool, error) {
				return cloudService.ReconcileControlPlaneEndpointDeletion(ctx, clusterScope)
			},
		},
	}
	for _, step := range reconcileSequence {
		if requeue, err := step.reconcileFunc(); err != nil || requeue {
			if err != nil {
				err = fmt.Errorf("error in step %s: %w", step.name, err)
			}

			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
		}
	}
	controllerutil.RemoveFinalizer(clusterScope.IonosCluster, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

func (r *IonosCloudClusterReconciler) checkRequestStatus(
	ctx context.Context, clusterScope *scope.ClusterScope, cloudService *cloud.Service,
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
					// remove the request from the status and patch the cluster
					ionosCluster.Status.CurrentClusterRequest = nil
					return nil
				},
			)
		}
	}
	return requeue, retErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *IonosCloudClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.IonosCloudCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Watches(&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(
				util.ClusterToInfrastructureMapFunc(ctx, infrav1.GroupVersion.WithKind(infrav1.IonosCloudClusterKind), r.Client, &infrav1.IonosCloudCluster{})),
			builder.WithPredicates(predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}
