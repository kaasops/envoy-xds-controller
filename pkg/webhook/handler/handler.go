package handler

import (
	"context"
	"fmt"

	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Handler struct {
	client.Client
	Config *config.Config
}

func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	fmt.Println("LOL")
	return admission.Response{}
}
