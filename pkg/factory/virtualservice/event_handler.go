package virtualservice

import (
	"context"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var enqueueLog = log.Log.WithName("eventhandler").WithName("EnqueueRequestForVirtualService")

type EnqueueRequestForVirtualService struct {
	Client client.Client
}
type empty struct{}

// Create implements EventHandler.
func (e *EnqueueRequestForVirtualService) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}

	vs, ok := evt.Object.(*v1alpha1.VirtualService)
	if !ok {
		enqueueLog.Error(nil, "Object is not a VirtualService", "event", evt)
		return
	}

	if err := v1alpha1.FillFromTemplateIfNeeded(ctx, e.Client, vs); err != nil {
		enqueueLog.Error(err, "failed to fill virtualService from template", "virtualService", evt.Object.GetName())
		return
	}

	req := types.NamespacedName{
		Name:      vs.GetListener(),
		Namespace: vs.GetNamespace(),
	}

	checkResult, err := vs.CheckHash()
	if err != nil {
		enqueueLog.Error(err, "failed to get virtualService hash", "virtualService", evt.Object.GetName())
		q.Add(reconcile.Request{NamespacedName: req})
	}

	if checkResult {
		enqueueLog.Info("VirtualService has no changes, skip event", "virtualService", evt.Object.GetName())
		return
	}

	q.Add(reconcile.Request{NamespacedName: req})
}

// Update implements EventHandler.
func (e *EnqueueRequestForVirtualService) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectNew == nil && evt.ObjectOld == nil {
		enqueueLog.Error(nil, "UpdateEvent received with no metadata", "event", evt)
		return
	}

	reqs := map[reconcile.Request]empty{}

	if evt.ObjectNew != nil {
		newVS, ok := evt.ObjectNew.(*v1alpha1.VirtualService)
		if ok {

			if err := v1alpha1.FillFromTemplateIfNeeded(ctx, e.Client, newVS); err != nil {
				enqueueLog.Error(err, "failed to fill virtualService from template", "virtualService", evt.ObjectNew.GetName())
				return
			}

			req := reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      newVS.GetListener(),
				Namespace: newVS.GetNamespace(),
			}}
			_, ok := reqs[req]
			if !ok {
				q.Add(req)
				reqs[req] = empty{}
			}
		} else {
			enqueueLog.Error(nil, "Object is not a VirtualService", "event", evt)
		}
	}

	if evt.ObjectOld != nil {
		oldVS, ok := evt.ObjectOld.(*v1alpha1.VirtualService)

		if ok {

			if err := v1alpha1.FillFromTemplateIfNeeded(ctx, e.Client, oldVS); err != nil {
				enqueueLog.Error(err, "failed to fill virtualService from template", "virtualService", evt.ObjectOld.GetName())
				return
			}

			req := reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      oldVS.GetListener(),
				Namespace: oldVS.GetNamespace(),
			}}
			_, ok := reqs[req]
			if !ok {
				q.Add(req)
				reqs[req] = empty{}
			}
		} else {
			enqueueLog.Error(nil, "Object is not a VirtualService", "event", evt)
		}
	}
}

// Delete implements EventHandler.
func (e *EnqueueRequestForVirtualService) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		enqueueLog.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		return
	}
	vs, ok := evt.Object.(*v1alpha1.VirtualService)
	if !ok {
		enqueueLog.Error(nil, "Object is not a VirtualService", "event", evt)
		return
	}
	if err := v1alpha1.FillFromTemplateIfNeeded(ctx, e.Client, vs); err != nil {
		enqueueLog.Error(err, "failed to fill virtualService from template", "virtualService", evt.Object.GetName())
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      vs.GetListener(),
		Namespace: vs.GetNamespace(),
	}})
}

// Generic implements EventHandler.
func (e *EnqueueRequestForVirtualService) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		enqueueLog.Error(nil, "GenericEvent received with no metadata", "event", evt)
		return
	}
	vs, ok := evt.Object.(*v1alpha1.VirtualService)
	if !ok {
		enqueueLog.Error(nil, "Object is not a VirtualService", "event", evt)
		return
	}
	if err := v1alpha1.FillFromTemplateIfNeeded(ctx, e.Client, vs); err != nil {
		enqueueLog.Error(err, "failed to fill virtualService from template", "virtualService", evt.Object.GetName())
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      vs.GetListener(),
		Namespace: vs.GetNamespace(),
	}})
}

var predicateLog = log.Log.WithName("eventhandler").WithName("GenerationOrMetadaChangedPredicate")

type GenerationOrMetadataChangedPredicate struct {
	predicate.Funcs
}

func (GenerationOrMetadataChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		predicateLog.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		predicateLog.Error(nil, "Update event has no new object to update", "event", e)
		return false
	}

	if !reflect.DeepEqual(e.ObjectNew.GetAnnotations(), e.ObjectOld.GetAnnotations()) {
		return true
	}

	if !reflect.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels()) {
		return true
	}

	if e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration() {
		return true
	}

	return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
}
