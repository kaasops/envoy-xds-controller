package grpcapi

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/util/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/util/v1/utilv1connect"
	virtual_service_templatev1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
)

type UtilsService struct {
	store *store.Store
	utilv1connect.UnimplementedUtilsServiceHandler
}

func NewUtilsService(s *store.Store) *UtilsService {
	return &UtilsService{store: s}
}

func (s *UtilsService) VerifyDomains(_ context.Context, req *connect.Request[v1.VerifyDomainsRequest]) (*connect.Response[v1.VerifyDomainsResponse], error) {
	results := make([]*v1.DomainVerificationResult, 0, len(req.Msg.Domains))
	for _, domain := range req.Msg.Domains {
		res := verifyDomainFromSecrets(domain, s.store.MapDomainSecrets())
		results = append(results, res)
	}
	return connect.NewResponse(&v1.VerifyDomainsResponse{Results: results}), nil
}

func verifyDomainFromSecrets(domain string, secrets map[string]corev1.Secret) *v1.DomainVerificationResult {
	result := &v1.DomainVerificationResult{Domain: domain}
	var matchedSecret *corev1.Secret
	var matchedByWildcard bool

	if secret, ok := secrets[domain]; ok {
		matchedSecret = &secret
	} else {
		parts := strings.Split(domain, ".")
		for i := 1; i < len(parts)-1; i++ {
			wildcard := "*." + strings.Join(parts[i:], ".")
			if secret, ok := secrets[wildcard]; ok {
				matchedSecret = &secret
				matchedByWildcard = true
				break
			}
		}
	}

	if matchedSecret == nil {
		result.Error = "no matching certificate found"
		return result
	}

	crtBytes, ok := matchedSecret.Data["tls.crt"]
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
	result.MatchedByWildcard = matchedByWildcard

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
