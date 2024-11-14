package middlewares

import (
	"context"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gin-gonic/gin"
)

const AvailableNodeIDs = "available_node_ids"

type Auth struct {
	verifier *oidc.IDTokenVerifier
	aclStore *aclStore
	devMode  bool
}

type ACL struct {
	Group   string   `json:"group"`
	NodeIDs []string `json:"nodeIds"`
}

type aclStore struct {
	enabled        bool
	fullAccess     map[string]struct{}
	nodeIDsByGroup map[string][]string
}

func NewAuth(issuerURL, clientID string, acl map[string][]string, devMode bool) (*Auth, error) {
	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	return &Auth{
		verifier: verifier,
		aclStore: newACLStore(acl),
		devMode:  devMode,
	}, nil
}

func (m *Auth) HandlerFunc(c *gin.Context) {
	respMessage := func(message, details string, err error) gin.H {
		resp := gin.H{"message": message}
		if m.devMode {
			if err != nil {
				resp["error"] = err.Error()
			}
			if details != "" {
				resp["details"] = details
			}
		}
		return resp
	}

	token, err := request.OAuth2Extractor.ExtractToken(c.Request)
	if err != nil {
		c.JSON(401, respMessage("Unauthorized", "failed to extract token", err))
		c.Abort()
		return
	}
	idToken, err := m.verifier.Verify(c.Request.Context(), token)
	if err != nil {
		c.JSON(401, respMessage("Unauthorized", "failed to verify token", err))
		c.Abort()
		return
	}

	var claims struct {
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.JSON(401, respMessage("Unauthorized", "failed to get claims", err))
		c.Abort()
		return
	}

	if m.aclStore.enabled {
		fullAccess := false
		availableNodeIds := make(map[string]struct{})
		for _, group := range claims.Groups {
			if _, ok := m.aclStore.fullAccess[group]; ok {
				fullAccess = true
				break
			}
			if nodeIds, ok := m.aclStore.nodeIDsByGroup[group]; ok {
				for _, nodeId := range nodeIds {
					availableNodeIds[nodeId] = struct{}{}
				}
			}
		}

		switch {
		case fullAccess:
		case len(availableNodeIds) > 0:
			c.Set(AvailableNodeIDs, availableNodeIds)
		default:
			c.JSON(401, respMessage("Unauthorized", "not available node ids", err))
			c.Abort()
			return
		}
	}

	c.Next()
}

func newACLStore(acl map[string][]string) *aclStore {
	store := &aclStore{}
	if len(acl) == 0 {
		return store
	}
	store.enabled = true
	store.fullAccess = make(map[string]struct{})
	store.nodeIDsByGroup = make(map[string][]string)

LOOP:
	for group, nodeIDs := range acl {
		for _, nodeID := range nodeIDs {
			if nodeID == "*" {
				store.fullAccess[group] = struct{}{}
				continue LOOP
			}
		}
		store.nodeIDsByGroup[group] = nodeIDs
	}
	return store
}
