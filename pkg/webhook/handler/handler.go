package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
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

var (
	ErrWrongGroup = errors.New("validator works only for resources within the envoy.kaasops.io group")

	ErrUnmarshal = errors.New("can't unmarshal resource")
)

func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Check resource Group
	if req.AdmissionRequest.Kind.Group != "envoy.kaasops.io" {
		return admission.Errored(http.StatusInternalServerError, ErrWrongGroup)
	}

	switch res := req.AdmissionRequest.Kind.Kind; res {
	case "VirtualService":
		if req.Operation != admissionv1.Delete {
			vs := &v1alpha1.VirtualService{}
			if err := json.Unmarshal(req.Object.Raw, vs); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := vs.Validate(ctx, h.Config, h.Client, h.DiscoveryClient); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "Listener":
		l := &v1alpha1.Listener{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, l); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := l.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, l); err != nil {
				if !api_errors.IsNotFound(err) {
					return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
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
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}

			if err := c.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "HttpFilter":
		hf := &v1alpha1.HttpFilter{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, hf); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := hf.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, hf); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := hf.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "Route":
		r := &v1alpha1.Route{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, r); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := r.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, r); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := r.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	case "AccessLogConfig":
		al := &v1alpha1.AccessLogConfig{}

		if req.Operation == admissionv1.Delete {
			if err := json.Unmarshal(req.OldObject.Raw, al); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}
			if err := al.ValidateDelete(ctx, h.Client); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else {
			if err := json.Unmarshal(req.Object.Raw, al); err != nil {
				return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
			}

			if err := al.Validate(ctx); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	}

	return admission.Allowed("")
}
