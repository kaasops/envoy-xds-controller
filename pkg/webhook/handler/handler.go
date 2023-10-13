package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Handler struct {
	Unmarshaler protojson.UnmarshalOptions
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
		vs := &v1alpha1.VirtualService{}
		if err := json.Unmarshal(req.Object.Raw, vs); err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
		}

		if err := vs.Validate(ctx, h.Unmarshaler); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	case "Listener":
		l := &v1alpha1.Listener{}
		if err := json.Unmarshal(req.Object.Raw, l); err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("%w. %w", ErrUnmarshal, err))
		}

		if err := l.Validate(ctx, h.Unmarshaler); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

	}

	return admission.Allowed("All ok")
}
