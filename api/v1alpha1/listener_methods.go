/*
Copyright 2023.

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

package v1alpha1

import (
	"context"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (l *Listener) SetError(ctx context.Context, cl client.Client, msg string) error {
	if l.messageAlredySet(msg) {
		return nil
	}

	l.Status.Message = ptr.To(msg)
	l.Status.Valid = ptr.To(false)

	// TODO: Get all linked VirtualServices and update status to false

	return cl.Status().Update(ctx, l.DeepCopy())
}

func (l *Listener) SetValidWithMessage(ctx context.Context, cl client.Client, msg string) error {
	if l.messageAlredySet(msg) {
		return nil
	}

	l.Status.Message = ptr.To(msg)
	l.Status.Valid = ptr.To(true)

	return cl.Status().Update(ctx, l.DeepCopy())
}

func (l *Listener) SetValid(ctx context.Context, cl client.Client) error {
	if l.validAlredySet() {
		return nil
	}

	l.Status.Valid = ptr.To(true)
	l.Status.Message = nil

	return cl.Status().Update(ctx, l.DeepCopy())
}

func (l *Listener) messageAlredySet(msg string) bool {
	if l.Status.Message != nil && *l.Status.Message == msg {
		return true
	}

	return false
}

func (l *Listener) validAlredySet() bool {
	if l.Status.Valid != nil && *l.Status.Valid {
		return true
	}

	return false
}
