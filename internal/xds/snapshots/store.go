package snapshots

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

// Secret keys: the frozen nodes as JSON, and the highest version a freeze or rollback has served.
const (
	freezeKey  = "frozen.json"
	versionKey = "version"
)

// SecretStore keeps the frozen nodes in a Secret of the controller namespace. A Secret, not a ConfigMap:
// what it holds decides what a node is served after a restart and names the operators, and the
// controller manages Secrets of its namespace already. It carries no envoy.kaasops.io/secret-type label,
// so the builder never serves it.
type SecretStore struct {
	// Reader reads past the manager cache.
	Reader    client.Reader
	Writer    client.Writer
	Namespace string
	Name      string
}

// Load returns the frozen nodes and the version floor; a missing Secret means none and zero.
func (s *SecretStore) Load(ctx context.Context) (map[string]wrapped.Freeze, int64, error) {
	var secret corev1.Secret
	err := s.Reader.Get(ctx, s.key(), &secret)
	if apierrors.IsNotFound(err) {
		return map[string]wrapped.Freeze{}, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("read frozen nodes from Secret %s: %w", s.key(), err)
	}
	frozen := map[string]wrapped.Freeze{}
	if data := secret.Data[freezeKey]; len(data) > 0 {
		if err := json.Unmarshal(data, &frozen); err != nil {
			return nil, 0, fmt.Errorf("parse frozen nodes in Secret %s: %w", s.key(), err)
		}
	}
	var floor int64
	if data := secret.Data[versionKey]; len(data) > 0 {
		if floor, err = strconv.ParseInt(string(data), 10, 64); err != nil {
			return nil, 0, fmt.Errorf("parse version in Secret %s: %w", s.key(), err)
		}
	}
	return frozen, floor, nil
}

// Save replaces the frozen nodes and the version floor.
func (s *SecretStore) Save(ctx context.Context, frozen map[string]wrapped.Freeze, versionFloor int64) error {
	data, err := json.Marshal(frozen)
	if err != nil {
		return err
	}
	values := map[string][]byte{freezeKey: data, versionKey: []byte(strconv.FormatInt(versionFloor, 10))}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var secret corev1.Secret
		err := s.Reader.Get(ctx, s.key(), &secret)
		if apierrors.IsNotFound(err) {
			secret = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: s.Namespace, Name: s.Name},
				Type:       corev1.SecretTypeOpaque,
				Data:       values,
			}
			return s.Writer.Create(ctx, &secret)
		}
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		for k, v := range values {
			secret.Data[k] = v
		}
		return s.Writer.Update(ctx, &secret)
	})
	if err != nil {
		return fmt.Errorf("save frozen nodes to Secret %s: %w", s.key(), err)
	}
	return nil
}

func (s *SecretStore) key() types.NamespacedName {
	return types.NamespacedName{Namespace: s.Namespace, Name: s.Name}
}
