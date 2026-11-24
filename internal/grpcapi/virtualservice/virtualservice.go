package virtualservice

import (
	"context"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1/virtual_servicev1connect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualServiceStore struct {
	store        store.Store
	client       client.Client
	apiReader    client.Reader
	targetNs     string
	cacheUpdater *updater.CacheUpdater // for clone store only
	virtual_servicev1connect.UnimplementedVirtualServiceStoreServiceHandler
}

func NewVirtualServiceStore(s store.Store, c client.Client, apiReader client.Reader, targetNs string, updater *updater.CacheUpdater) *VirtualServiceStore {
	return &VirtualServiceStore{
		store:        s,
		client:       c,
		apiReader:    apiReader,
		targetNs:     targetNs,
		cacheUpdater: updater,
	}
}

const uidIndex = "metadata.uid"

// IndexByUID lets Get resolve the same watched Kubernetes objects that List returns.
func IndexByUID(ctx context.Context, indexer client.FieldIndexer) error {
	return indexer.IndexField(ctx, &api.VirtualService{}, uidIndex, indexUID)
}

func indexUID(obj client.Object) []string {
	return []string{string(obj.GetUID())}
}
