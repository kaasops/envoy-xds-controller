package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/client-go/util/retry"
	"os"
	"path/filepath"

	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyManifest(c client.Client, manifestPath string, ns string) error {
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		obj.SetNamespace(ns)

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := c.Create(context.Background(), obj)
			if err != nil {
				if api_errors.IsAlreadyExists(err) {
					var current unstructured.Unstructured
					current.SetGroupVersionKind(obj.GroupVersionKind())
					err = c.Get(context.TODO(), types.NamespacedName{
						Name:      obj.GetName(),
						Namespace: obj.GetNamespace(),
					}, &current)
					if err != nil {
						return fmt.Errorf("failed to get current version of resource: %v", err)
					}
					currentBytes, err := current.MarshalJSON()
					if err != nil {
						return fmt.Errorf("failed to marshal current version of resource: %v", err)
					}
					obj.SetResourceVersion(current.GetResourceVersion())
					obj.SetGeneration(current.GetGeneration())
					obj.SetResourceVersion(current.GetResourceVersion())
					obj.SetUID(current.GetUID())
					modifiedBytes, err := obj.MarshalJSON()
					if err != nil {
						return fmt.Errorf("failed to marshal modified version of resource: %v", err)
					}
					patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(currentBytes, modifiedBytes, []byte{})
					if err != nil {
						return fmt.Errorf("failed to create two-way merge patch: %v", err)
					}
					err = c.Patch(context.Background(), &current, client.RawPatch(types.MergePatchType, patch))
					if err != nil {
						return fmt.Errorf("failed to patch resource: %v", err)
					}
					return nil
				}
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func ApplyManifests(c client.Client, manifests []string, ns string) error {
	// TODO: Add GOROUTINES
	for _, manifest := range manifests {
		err := ApplyManifest(c, manifest, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func ApplyManifestsFromPath(c client.Client, manifestsPath string, ns string) error {
	files, err := os.ReadDir(manifestsPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			return errors.New("unexpected directory in base manifests")
		}

		filename := filepath.Join(manifestsPath, file.Name())

		err = ApplyManifest(c, filename, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func CleanupManifest(c client.Client, manifestPath string, ns string) error {
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		obj.SetNamespace(ns)

		if err := c.Delete(context.Background(), obj); err != nil {
			return err
		}
	}

	return nil
}

func CleanupManifests(c client.Client, manifests []string, ns string) error {
	for _, manifest := range manifests {
		err := CleanupManifest(c, manifest, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func CleanupManifestsFromPath(c client.Client, manifestsPath string, ns string) error {
	files, err := os.ReadDir(manifestsPath)
	if err != nil {
		return err
	}
	slices.Reverse(files)

	for _, file := range files {
		if file.IsDir() {
			return errors.New("unexpected directory in base manifests")
		}

		filename := filepath.Join(manifestsPath, file.Name())

		err = CleanupManifest(c, filename, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func CreateSecretInNamespace(
	suite *TestSuite,
	secretPath, secretNamespaceName string,
) error {
	// If Namespace for secret not set - use suite Namespace
	if secretNamespaceName != suite.Namespace {
		// Create Namespace for secret if set special
		err := suite.Client.Create(context.TODO(), &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Namespace",
				"metadata": map[string]interface{}{
					"name": secretNamespaceName,
				},
			},
		})
		if err != nil {
			if !api_errors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	// Create secret with Certificate in special Namespace
	err := ApplyManifest(
		suite.Client,
		secretPath,
		secretNamespaceName,
	)
	if err != nil {
		return err
	}

	return nil
}

func CleanupNamespace(
	ctx context.Context,
	cl client.Client,
	namespaceName string,
) error {
	return cl.Delete(ctx, &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": namespaceName,
			},
		},
	})
}

// func CleanupSecret(
// 	ctx context.Context,
// 	secretPath, secretNamespaceName string,
// 	cl client.Client,
// ) error {
// 	return CleanupManifest(
// 		cl,
// 		secretPath,
// 		secretNamespaceName,
// 	)
// }
