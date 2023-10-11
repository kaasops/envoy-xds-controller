package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Handler struct {
	client.Client
	Unmarshaler *protojson.UnmarshalOptions
	Config      *config.Config
}

func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Check resource Group
	if req.AdmissionRequest.Kind.Group != "envoy.kaasops.io" {
		return admission.Errored(http.StatusInternalServerError, errors.New("validator only works for resources within the envoy.kaasops.io group"))
	}

	switch res := req.AdmissionRequest.Kind.Kind; res {
	case "VirtualService":
		vs := &v1alpha1.VirtualService{}
		if err := json.Unmarshal(req.Object.Raw, vs); err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.New("validator error unmarshal Virtual Service"))
		}

		if err := vs.Validate(h.Client, h.Unmarshaler); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

	}

	return admission.Allowed("All ok")
}
