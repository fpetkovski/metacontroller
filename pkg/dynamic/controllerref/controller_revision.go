/*
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllerref

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/klog/v2"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/utils/pointer"
	"metacontroller.io/pkg/apis/metacontroller/v1alpha1"
	k8s "metacontroller.io/pkg/third_party/kubernetes"
)

type ControllerRevisionManager struct {
	k8s.BaseControllerRefManager
	parentKind schema.GroupVersionKind
	k8sClient  client.Client
}

func NewControllerRevisionManager(k8sClient client.Client, parent metav1.Object, selector labels.Selector, parentKind schema.GroupVersionKind, canAdopt func() error) *ControllerRevisionManager {
	return &ControllerRevisionManager{
		BaseControllerRefManager: k8s.BaseControllerRefManager{
			Controller:   parent,
			Selector:     selector,
			CanAdoptFunc: canAdopt,
		},
		parentKind: parentKind,
		k8sClient:  k8sClient,
	}
}

func (m *ControllerRevisionManager) ClaimControllerRevisions(children []v1alpha1.ControllerRevision) ([]*v1alpha1.ControllerRevision, error) {
	var claimed []*v1alpha1.ControllerRevision
	var errlist []error

	match := func(obj metav1.Object) bool {
		return m.Selector.Matches(labels.Set(obj.GetLabels()))
	}
	adopt := func(obj metav1.Object) error {
		return m.adoptControllerRevision(obj.(*v1alpha1.ControllerRevision))
	}
	release := func(obj metav1.Object) error {
		return m.releaseControllerRevision(obj.(*v1alpha1.ControllerRevision))
	}

	for _, child := range children {
		ok, err := m.ClaimObject(&child, match, adopt, release)
		if err != nil {
			errlist = append(errlist, err)
			continue
		}
		if ok {
			claimed = append(claimed, &child)
		}
	}
	return claimed, utilerrors.NewAggregate(errlist)
}

func (m *ControllerRevisionManager) adoptControllerRevision(obj *v1alpha1.ControllerRevision) error {
	if err := m.CanAdopt(); err != nil {
		return fmt.Errorf("can't adopt ControllerRevision %v/%v (%v): %v", obj.GetNamespace(), obj.GetName(), obj.GetUID(), err)
	}
	klog.InfoS("Adopting ControllerRevision", "kind", m.parentKind.Kind, "controller", klog.KRef(m.Controller.GetNamespace(), m.Controller.GetName()), "object", klog.KObj(obj))
	controllerRef := metav1.OwnerReference{
		APIVersion:         m.parentKind.GroupVersion().String(),
		Kind:               m.parentKind.Kind,
		Name:               m.Controller.GetName(),
		UID:                m.Controller.GetUID(),
		Controller:         pointer.BoolPtr(true),
		BlockOwnerDeletion: pointer.BoolPtr(true),
	}

	// We can't use strategic merge patch because we want this to work with custom resources.
	// We can't use merge patch because that would replace the whole list.
	// We can't use JSON patch ops because that wouldn't be idempotent.
	// The only option is GET/PUT with ResourceVersion.
	_, err := m.UpdateWithRetries(obj, func(obj *v1alpha1.ControllerRevision) bool {
		ownerRefs := addOwnerReference(obj.GetOwnerReferences(), controllerRef)
		obj.SetOwnerReferences(ownerRefs)
		return true
	})
	return err
}

func (m *ControllerRevisionManager) releaseControllerRevision(obj *v1alpha1.ControllerRevision) error {
	klog.InfoS("Releasing ControllerRevision", "kind", m.parentKind.Kind, "controller", klog.KRef(m.Controller.GetNamespace(), m.Controller.GetName()), "object", klog.KObj(obj))
	_, err := m.UpdateWithRetries(obj, func(obj *v1alpha1.ControllerRevision) bool {
		ownerRefs := removeOwnerReference(obj.GetOwnerReferences(), m.Controller.GetUID())
		obj.SetOwnerReferences(ownerRefs)
		return true
	})
	if apierrors.IsNotFound(err) || apierrors.IsGone(err) {
		// If the original object is gone, that's fine because we're giving up on this child anyway.
		return nil
	}
	return err
}

func (c *ControllerRevisionManager) UpdateWithRetries(orig *v1alpha1.ControllerRevision, updateFn func(*v1alpha1.ControllerRevision) bool) (*v1alpha1.ControllerRevision, error) {
	key := types.NamespacedName{
		Namespace: orig.GetNamespace(),
		Name:      orig.GetName(),
	}
	var current v1alpha1.ControllerRevision
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := c.k8sClient.Get(context.Background(), key, &current)
		if err != nil {
			return err
		}
		if current.GetUID() != orig.GetUID() {
			return apierrors.NewGone(fmt.Sprintf("can't update ControllerRevision %v/%v: original object is gone: got uid %v, want %v", orig.GetNamespace(), orig.GetName(), current.GetUID(), orig.GetUID()))
		}
		if changed := updateFn(&current); !changed {
			// There's nothing to do.
			return nil
		}
		err = c.k8sClient.Update(context.Background(), &current)
		return err
	})

	return &current, err
}
