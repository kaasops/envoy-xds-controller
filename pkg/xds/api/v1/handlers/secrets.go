package handlers

import (
	"net/url"
	"slices"

	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/gin-gonic/gin"
)

type GetSecretsResponse struct {
	Secrets []*tlsv3.Secret `json:"secrets"`
}

func (h *handler) getSecrets(ctx *gin.Context) {
	params, err := h.getParamsFoSecretRequests(ctx.Request.URL.Query())
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check node_id exist in cache
	nodeIDs := h.cache.GetNodeIDs()
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

	secrets, err := h.getSecretsAll(params[nodeIDParamName][0])
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
