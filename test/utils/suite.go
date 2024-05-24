package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestSuite struct {
	Client    client.Client
	Namespace string
	SkipTests []string
}

type TestCase struct {
	ShortName          string
	Description        string
	Manifests          []string
	ApplyErrorContains string
	Test               func(t *testing.T, suite *TestSuite)
}

func (tc *TestCase) Run(t *testing.T, suite *TestSuite) {
	t.Helper()

	// Skip test if it is in the SkipTests list
	if suite.SkipTests != nil {
		for _, skip := range suite.SkipTests {
			if tc.ShortName == skip {
				t.Skipf("Skipping test %s", tc.ShortName)
			}
		}
	}

	// Apply manifests
	for _, manifest := range tc.Manifests {
		err := ApplyManifest(suite.Client, manifest, suite.Namespace)
		if err != nil {
			if tc.ApplyErrorContains == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.ApplyErrorContains)
			}
		}
	}
	// Cleanup manifests
	defer func() {
		CleanupManifests(suite.Client, tc.Manifests, suite.Namespace)
	}()

	tc.Test(t, suite)
}
