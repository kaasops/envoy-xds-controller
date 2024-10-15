package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	admissionv1 "k8s.io/api/admission/v1"
)

type Handler struct {
	Config          *config.Config
	Client          client.Client
	DiscoveryClient *discovery.DiscoveryClient
}

// var (
// 	ErrUnmarshal          = errors.New("can't unmarshal resource")
// 	ErrGetVirtualServices = errors.New("cannot get Virtual Services")
// )

func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Check if resources work in not-control-namespaces
	watchNamespaces := h.Config.GetWatchNamespaces()
	if watchNamespaces != nil {
		if !slices.Contains(watchNamespaces, req.Namespace) {
			return admission.Allowed("")
		}
	}

	switch res := req.AdmissionRequest.Kind.Kind; res {
	case "VirtualService":
		if req.Operation != admissionv1.Delete {
			vs := &v1alpha1.VirtualService{}
			if err := json.Unmarshal(req.Object.Raw, vs); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
			if err := vs.Validate(ctx, h.Config, h.Client, h.DiscoveryClient); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "Listener":
		l := &v1alpha1.Listener{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, l); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
			if err := l.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, l); err != nil {
				if !api_errors.IsNotFound(err) {
					return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
				}
			}
			if err := l.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "Cluster":
		c := &v1alpha1.Cluster{}

		if req.Operation != admissionv1.Delete {
			if err := json.Unmarshal(req.Object.Raw, c); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}

			if err := c.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "HttpFilter":
		hf := &v1alpha1.HttpFilter{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, hf); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
			if err := hf.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, hf); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
			if err := hf.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "Route":
		r := &v1alpha1.Route{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, r); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
			if err := r.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, r); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v*. %w", errors.UnmarshalMessage, err))
			}
			if err := r.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "AccessLogConfig":
		al := &v1alpha1.AccessLogConfig{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, al); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
			if err := al.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, al); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}

			if err := al.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "Secret":
		if req.Operation == admissionv1.Delete {
			// If certificate in secret used in any VirtualService - cannot delete!
			virtualServices := &v1alpha1.VirtualServiceList{}

			if err := h.Client.List(ctx, virtualServices); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.GetFromKubernetesMessage, err))
			}

			for _, vs := range virtualServices.Items {
				for _, us := range vs.Status.UsedSecrets {
					if us.Name == req.Name && *us.Namespace == req.Namespace {
						return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. It used in Virtual Service %v/%v", errors.DeleteInKubernetesMessage, vs.Namespace, vs.Name))
					}
				}
			}

		}
	// TODO: Add check for double certificates
	case "Policy":
		policy := &v1alpha1.Policy{}

		if req.Operation == admissionv1.Delete {

			if err := json.Unmarshal(req.OldObject.Raw, policy); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}

			virtualServices := &v1alpha1.VirtualServiceList{}

			if err := h.Client.List(ctx, virtualServices); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.GetFromKubernetesMessage, err))
			}

			for _, vs := range virtualServices.Items {
				if vs.Spec.RBAC == nil || len(vs.Spec.RBAC.AdditionalPolicies) == 0 {
					continue
				}

				for _, p := range vs.Spec.RBAC.AdditionalPolicies {
					if p.Name == req.Name &&
						((p.Namespace != nil && *p.Namespace == req.Namespace) || req.Namespace == vs.Namespace) {
						return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. It used in Virtual Service %v/%v", errors.DeleteInKubernetesMessage, vs.Namespace, vs.Name))
					}
				}
			}

		} else {

			if err := json.Unmarshal(req.Object.Raw, policy); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}

			if err := policy.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "VirtualServiceTemplate":
		vst := &v1alpha1.VirtualServiceTemplate{}

		if req.Operation == admissionv1.Delete {

			if err := json.Unmarshal(req.OldObject.Raw, vst); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}

			virtualServices := &v1alpha1.VirtualServiceList{}

			if err := h.Client.List(ctx, virtualServices); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.GetFromKubernetesMessage, err))
			}

			for _, vs := range virtualServices.Items {
				if vs.Spec.Template == nil {
					continue
				}

				if vs.Spec.Template.Name == req.Name &&
					((vs.Spec.Template.Namespace != nil && *vs.Spec.Template.Namespace == req.Namespace) || req.Namespace == vs.Namespace) {
					return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. It used in Virtual Service %v/%v", errors.DeleteInKubernetesMessage, vs.Namespace, vs.Name))
				}
			}

		} else {
			if err := json.Unmarshal(req.Object.Raw, vst); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%v. %w", errors.UnmarshalMessage, err))
			}
		}
	}

	return admission.Allowed("")
}
