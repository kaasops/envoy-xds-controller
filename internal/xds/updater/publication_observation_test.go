package updater

import (
	"testing"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestPublicationQueueAndObjectIdentity(t *testing.T) {
	u := NewCacheUpdater(cache.NewSnapshotCache(), store.New())
	vs := &api.VirtualService{ObjectMeta: metav1.ObjectMeta{
		Namespace: "default", Name: "example", UID: "first", Generation: 1,
	}}
	u.ApplyVirtualService(t.Context(), vs)
	var keys []helpers.NamespacedName
	key := helpers.NamespacedName{Namespace: "default", Name: "example"}
	u.WatchPublicationStatus(func(nn helpers.NamespacedName) { keys = append(keys, nn) })
	require.Equal(t, []helpers.NamespacedName{key}, keys)
	require.Error(t, u.RebuildSnapshots(t.Context()))
	require.Len(t, keys, 1, "identical result must not retrigger a Kubernetes status write")

	newer := vs.DeepCopy()
	newer.Generation++
	_, known := u.PublicationStatusFor(newer)
	require.False(t, known)
	u.ApplyVirtualService(t.Context(), newer)
	_, known = u.PublicationStatusFor(newer)
	require.True(t, known)
	require.Len(t, keys, 2)

	replacement := newer.DeepCopy()
	replacement.UID = "replacement"
	_, known = u.PublicationStatusFor(replacement)
	require.False(t, known)
	u.ApplyVirtualService(t.Context(), replacement)
	_, known = u.PublicationStatusFor(newer)
	require.False(t, known)
	require.Len(t, keys, 3)

	require.NoError(t, u.DeleteVirtualService(t.Context(), types.NamespacedName(key)))
	_, known = u.PublicationStatusFor(replacement)
	require.False(t, known)
}
