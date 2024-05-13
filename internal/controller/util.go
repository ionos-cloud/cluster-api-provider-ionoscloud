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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	icc "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/client"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	defaultReconcileDuration = time.Second * 20
)

type serviceReconcileStep[T scope.Cluster | scope.Machine] struct {
	name string
	fn   func(context.Context, *T) (requeue bool, err error)
}

// withStatus is a helper function to handle the different request states
// and provides a callback function to execute when the request is done or failed.
func withStatus(
	status string,
	message string,
	log *logr.Logger,
	doneOrFailedCallback func() error,
) (requeue bool, err error) {
	switch status {
	case sdk.RequestStatusQueued, sdk.RequestStatusRunning:
		return true, nil
	case sdk.RequestStatusFailed:
		// log the error message
		log.Error(nil, "Request status indicates a failure", "message", message)
		fallthrough // we run the same logic as for status done
	case sdk.RequestStatusDone:
		// we don't requeue
		return false, doneOrFailedCallback()
	}

	return false, fmt.Errorf("unknown request status %s", status)
}

func createServiceFromCluster(
	ctx context.Context,
	c client.Client,
	cluster *infrav1.IonosCloudCluster,
	log logr.Logger,
) (*cloud.Service, error) {
	secretKey := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.CredentialsRef.Name,
	}

	var authSecret corev1.Secret
	if err := c.Get(ctx, secretKey, &authSecret); err != nil {
		return nil, err
	}

	if err := ensureSecretControlledByCluster(ctx, c, cluster, &authSecret); err != nil {
		return nil, err
	}

	token := string(authSecret.Data["token"])
	apiURL := string(authSecret.Data["apiURL"])
	caBundle := authSecret.Data["caBundle"]

	ionosClient, err := icc.NewClient(token, apiURL, caBundle)
	if err != nil {
		return nil, err
	}

	return cloud.NewService(ionosClient, log)
}

// ensureSecretControlledByCluster ensures that the secrets will contain a cluster-specific finalizer and an owner reference.
// The secret will be deleted automatically with its last owner.
func ensureSecretControlledByCluster(
	ctx context.Context, c client.Client,
	cluster *infrav1.IonosCloudCluster,
	secret *corev1.Secret,
) error {
	old := secret.DeepCopy()

	finalizerAdded := controllerutil.AddFinalizer(secret, fmt.Sprintf("%s/%s", infrav1.ClusterFinalizer, cluster.GetUID()))
	// We want to allow using the secret in multiple clusters.
	// Using owner references because Kubernetes only allows us to have one controller reference.
	if err := controllerutil.SetOwnerReference(cluster, secret, c.Scheme()); err != nil {
		return err
	}

	if finalizerAdded || !cmp.Equal(old.GetOwnerReferences(), secret.GetOwnerReferences()) {
		return c.Update(ctx, secret)
	}

	return nil
}

// removeCredentialsFinalizer removes the cluster-specific finalizer from the credentials secret.
func removeCredentialsFinalizer(ctx context.Context, c client.Client, cluster *infrav1.IonosCloudCluster) error {
	secretKey := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.CredentialsRef.Name,
	}
	var secret corev1.Secret

	if err := c.Get(ctx, secretKey, &secret); err != nil {
		// If the secret does not exist anymore, there is nothing we can do.
		return client.IgnoreNotFound(err)
	}

	controllerutil.RemoveFinalizer(&secret, fmt.Sprintf("%s/%s", infrav1.ClusterFinalizer, cluster.GetUID()))
	return c.Update(ctx, &secret)
}
