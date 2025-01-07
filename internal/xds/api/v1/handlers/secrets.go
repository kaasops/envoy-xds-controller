package handlers

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"slices"
	"strings"

	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/gin-gonic/gin"
)

const (
	TypeTLSCertificate = "tls_certificate"
	TypeGenericSecret  = "generic_secret"
)

type GetSecretsResponse struct {
	Secrets []*tlsv3.Secret `json:"secrets"`
}

// getSecrets retrieves the secrets for a specific node ID.
// @Summary Get secrets for a specific node ID.
// @Tags secret
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param secret_name query string false "Secret name" format(string) example("secret-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetSecretsResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/secrets [get]
func (h *handler) getSecrets(ctx *gin.Context) {
	params, err := h.getParamsFoSecretRequests(ctx.Request.URL.Query())
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check node_id exist in cache
	nodeIDs := h.getAvailableNodeIDs(ctx)
	if !slices.Contains(nodeIDs, params[nodeIDParamName][0]) {
		ctx.JSON(400, gin.H{"error": "node_id not found in cache", "node_id": params[nodeIDParamName][0]})
		return
	}

	var response GetSecretsResponse

	// If param name set, return only one route configuration
	if params[secretParamName][0] != "" {
		secret, err := h.getSecretByName(params[nodeIDParamName][0], params[secretParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		response.Secrets = []*tlsv3.Secret{secret}
		ctx.JSON(200, response)
		return
	}

	secrets, err := h.getSecretsAll(params[nodeIDParamName][0], true)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	response.Secrets = secrets
	ctx.JSON(200, response)
}

func (h *handler) getParamsFoSecretRequests(queryValues url.Values) (map[string][]string, error) {
	qParams := []getParam{
		{
			name:     nodeIDParamName,
			required: true,
			onlyOne:  true,
		},
		{
			name:     secretParamName,
			required: false,
			onlyOne:  true,
		},
	}

	params, err := h.getParams(queryValues, qParams)
	if err != nil {
		return nil, err
	}

	return params, nil
}

type secretResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Certs     []struct {
		SerialNumber string   `json:"serialNumber"`
		Subject      string   `json:"subject"`
		NotBefore    string   `json:"notBefore"`
		NotAfter     string   `json:"notAfter"`
		Issuer       string   `json:"issuer"`
		Raw          string   `json:"raw"`
		DNSNames     []string `json:"dnsNames"`
	} `json:"certs,omitempty"`
}

func (h *handler) getSecretByNamespacedName(ctx *gin.Context) {
	nodeIDs := h.getAvailableNodeIDs(ctx)
	namespace := ctx.Param("namespace")
	name := ctx.Param("name")

	response := secretResponse{
		Namespace: namespace,
		Name:      name,
	}

	notFound := true

	for _, nodeID := range nodeIDs {
		secrets, err := h.getSecretsAll(nodeID, false)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		for _, secret := range secrets {
			parts := strings.Split(secret.Name, "/")
			if len(parts) == 2 && parts[0] == namespace && parts[1] == name {
				response.Name = parts[1]
				response.Namespace = parts[0]

				switch secret.Type.(type) {
				case *tlsv3.Secret_GenericSecret:
					response.Type = TypeGenericSecret
				case *tlsv3.Secret_TlsCertificate:
					response.Type = TypeTLSCertificate
					certs, err := parsePEM(secret.GetTlsCertificate().CertificateChain.GetInlineBytes())
					if err != nil {
						ctx.JSON(500, gin.H{"error": err.Error()})
						return
					}
					for _, cert := range certs {
						response.Certs = append(response.Certs, struct {
							SerialNumber string   `json:"serialNumber"`
							Subject      string   `json:"subject"`
							NotBefore    string   `json:"notBefore"`
							NotAfter     string   `json:"notAfter"`
							Issuer       string   `json:"issuer"`
							Raw          string   `json:"raw"`
							DNSNames     []string `json:"dnsNames"`
						}{
							SerialNumber: cert.SerialNumber.String(),
							Subject:      cert.Subject.String(),
							NotBefore:    cert.NotBefore.String(),
							NotAfter:     cert.NotAfter.String(),
							Issuer:       cert.Issuer.String(),
							Raw:          string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})),
							DNSNames:     cert.DNSNames,
						},
						)
					}
				}
				notFound = false
				break
			}
		}
	}

	if notFound {
		ctx.JSON(404, nil)
		return
	}

	ctx.JSON(200, response)
}

func parsePEM(data []byte) ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0)
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		switch block.Type {
		case "CERTIFICATE":
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			certs = append(certs, cert)

		case "PRIVATE KEY":
			_, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown block type %q", block.Type)
		}
	}
	return certs, nil
}
