/*
Copyright 2020 The Kubernetes Authors.

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

package controllers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// The Cluster owner reference of an infrastructure template is set with the
// Cluster's current API version: an older one would make the topology
// controller, which sets the current one, patch the template on every
// reconcile.
func TestReconcileExternalReferenceKeepsTheClusterAPIVersion(t *testing.T) {
	cluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default", UID: "cluster-uid"}}
	template := &unstructured.Unstructured{}
	template.SetAPIVersion("infrastructure.cluster.x-k8s.io/v1beta2")
	template.SetKind("GenericInfrastructureMachineTemplate")
	template.SetName("template")
	template.SetNamespace("default")
	template.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: clusterv1.GroupVersion.String(),
		Kind:       "Cluster",
		Name:       cluster.Name,
		UID:        cluster.UID,
	}})

	c := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(template).Build()
	r := &KThreesControlPlaneReconciler{Client: c}
	ref := corev1.ObjectReference{
		APIVersion: template.GetAPIVersion(),
		Kind:       template.GetKind(),
		Name:       template.GetName(),
		Namespace:  template.GetNamespace(),
	}
	if err := r.reconcileExternalReference(context.Background(), cluster, ref); err != nil {
		t.Fatalf("reconcileExternalReference: %v", err)
	}

	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(template.GroupVersionKind())
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(template), got); err != nil {
		t.Fatalf("get template: %v", err)
	}
	refs := got.GetOwnerReferences()
	if len(refs) != 1 || refs[0].APIVersion != clusterv1.GroupVersion.String() || refs[0].UID != cluster.UID {
		t.Errorf("want one owner reference to the Cluster with %s, got %+v", clusterv1.GroupVersion, refs)
	}
}
