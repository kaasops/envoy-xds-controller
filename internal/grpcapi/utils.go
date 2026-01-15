package grpcapi

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/util/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/util/v1/utilv1connect"
	virtual_service_templatev1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UtilsService struct {
	store store.Store
	utilv1connect.UnimplementedUtilsServiceHandler
}

func NewUtilsService(s store.Store) *UtilsService {
	return &UtilsService{store: s}
}

// VerifyDomains checks if valid TLS certificates exist for the given domains.
// Uses the same wildcard fallback logic as the secrets builder: if exact cert is
// expired/unparseable, falls back to a valid wildcard certificate.
//
// Wildcard matching: Only immediate parent wildcard is checked.
// For example, "api.example.com" will match "*.example.com" but
// "sub.api.example.com" will only match "*.api.example.com" (not "*.example.com").
func (s *UtilsService) VerifyDomains(_ context.Context, req *connect.Request[v1.VerifyDomainsRequest]) (*connect.Response[v1.VerifyDomainsResponse], error) {
	results := make([]*v1.DomainVerificationResult, 0, len(req.Msg.Domains))
	for _, domain := range req.Msg.Domains {
		res := s.verifyDomainWithFallback(domain)
		results = append(results, res)
	}
	return connect.NewResponse(&v1.VerifyDomainsResponse{Results: results}), nil
}

// verifyDomainWithFallback checks certificate for a domain using the same
// wildcard fallback logic as the secrets builder
func (s *UtilsService) verifyDomainWithFallback(domain string) *v1.DomainVerificationResult {
	result := &v1.DomainVerificationResult{Domain: domain}

	// Use store's wildcard fallback logic - empty namespace for domain-level check
	lookupResult := s.store.GetDomainSecretWithWildcardFallbackInfo(domain, "")

	if lookupResult.Secret == nil {
		result.Error = "no matching certificate found"
		return result
	}

	result.MatchedByWildcard = lookupResult.UsedWildcard

	// Parse and validate the matched certificate
	crtBytes, ok := lookupResult.Secret.Data["tls.crt"]
	if !ok {
		result.Error = "missing tls.crt in secret"
		return result
	}

	block, _ := pem.Decode(crtBytes)
	if block == nil {
		result.Error = "failed to parse PEM block"
		return result
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse certificate: %v", err)
		return result
	}

	result.Issuer = cert.Issuer.CommonName
	result.ExpiresAt = timestamppb.New(cert.NotAfter)

	// Note: lookupResult also contains FallbackReason and ExactValidity
	// for debugging purposes, but the proto doesn't support these fields yet.
	// When the proto is updated, this info can be exposed in the API response.

	if time.Now().After(cert.NotAfter) {
		result.Error = "certificate expired"
		return result
	}

	if err := cert.VerifyHostname(domain); err != nil {
		result.Error = fmt.Sprintf("certificate does not match domain: %v", err)
		return result
	}

	result.ValidCertificate = true
	return result
}

func ParseTemplateOptionModifier(modifier virtual_service_templatev1.TemplateOptionModifier) v1alpha1.Modifier {
	switch modifier {
	case virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_MERGE:
		return v1alpha1.ModifierMerge
	case virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_REPLACE:
		return v1alpha1.ModifierReplace
	case virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_DELETE:
		return v1alpha1.ModifierDelete
	}
	return ""
}

func ParseModifierToTemplateOption(modifier v1alpha1.Modifier) virtual_service_templatev1.TemplateOptionModifier {
	switch modifier {
	case v1alpha1.ModifierMerge:
		return virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_MERGE
	case v1alpha1.ModifierReplace:
		return virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_REPLACE
	case v1alpha1.ModifierDelete:
		return virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_DELETE
	}
	return virtual_service_templatev1.TemplateOptionModifier_TEMPLATE_OPTION_MODIFIER_UNSPECIFIED
}
