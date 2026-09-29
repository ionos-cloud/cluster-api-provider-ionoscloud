/*
Copyright 2026 IONOS Cloud.

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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

func TestReconcileDeleteRequeuesUntilOwningClusterIsDeleted(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, infrav1.AddToScheme(scheme))

	const namespace = "default"
	now := metav1.Now()

	ionosCluster := &infrav1.IonosCloudCluster{
		Name:              "test-cluster",
		Namespace:         namespace,
		UID:               "ionos-cluster-uid",
		DeletionTimestamp: &now,
		Finalizers:        []string{infrav1.ClusterFinalizer},
		Spec: infrav1.IonosCloudClusterSpec{
			CredentialsRef: corev1.LocalObjectReference{Name: "credentials"},
		},
	}
	credentialsFinalizer := fmt.Sprintf("%s/%s", infrav1.ClusterFinalizer, ionosCluster.GetUID())
	credentials := &corev1.Secret{
		Name:       "credentials",
		Namespace:  namespace,
		Finalizers: []string{credentialsFinalizer},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(credentials).Build()

	// The cached owning Cluster doesn't show the deletion yet.
	cluster := &clusterv1.Cluster{
		Name: "test-cluster", Namespace: namespace,
	}
	clusterScope, err := scope.NewCluster(scope.ClusterParams{
		Client:       cl,
		Cluster:      cluster,
		IonosCluster: ionosCluster,
		Locker:       locker.New(),
	})
	require.NoError(t, err)

	r := &IonosCloudClusterReconciler{Client: cl}
	ctx := context.Background()

	res, err := r.reconcileDelete(ctx, clusterScope, nil)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{RequeueAfter: defaultReconcileDuration}, res)
	require.True(t, controllerutil.ContainsFinalizer(ionosCluster, infrav1.ClusterFinalizer))
	requireSecretFinalizer(ctx, t, cl, credentials, credentialsFinalizer, true)

	// On the requeue, the cache shows the owning Cluster as deleted.
	deletingCluster := cluster.DeepCopy()
	deletingCluster.DeletionTimestamp = &now
	clusterScope.Cluster = deletingCluster

	res, err = r.reconcileDelete(ctx, clusterScope, nil)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, res)
	require.False(t, controllerutil.ContainsFinalizer(ionosCluster, infrav1.ClusterFinalizer))
	requireSecretFinalizer(ctx, t, cl, credentials, credentialsFinalizer, false)
}

func requireSecretFinalizer(
	ctx context.Context, t *testing.T, cl client.Client, secret *corev1.Secret, finalizer string, want bool,
) {
	t.Helper()

	got := &corev1.Secret{}
	require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(secret), got))
	require.Equal(t, want, controllerutil.ContainsFinalizer(got, finalizer))
}
