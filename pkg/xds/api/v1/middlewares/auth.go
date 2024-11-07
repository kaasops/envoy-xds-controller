package middlewares

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Auth struct {
	verifier *oidc.IDTokenVerifier
}

func NewAuth(issuerURL, clientID string) (*Auth, error) {
	ctx := oidc.ClientContext(context.TODO(), http.DefaultClient)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	return &Auth{
		verifier: verifier,
	}, nil
}

func (m *Auth) HandlerFunc(c *gin.Context) {
	token, err := request.OAuth2Extractor.ExtractToken(c.Request)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		c.Abort()
		return
	}
	_, err = m.verifier.Verify(c.Request.Context(), token)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		c.Abort()
		return
	}
	fmt.Println(token) // TODO:
	c.Next()
}
