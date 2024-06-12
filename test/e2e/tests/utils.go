package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func envoyIsReady(t *testing.T) bool {
	url := fmt.Sprintf("%s/%s", envoyAdminPannel(), "ready")

	req, err := http.Get(url)
	require.NoError(t, err)

	return req.StatusCode == http.StatusOK
}
