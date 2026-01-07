package utils

import (
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

var (
	// ClusterSlicePool предоставляет пул для слайсов кластеров
	ClusterSlicePool = sync.Pool{
		New: func() interface{} {
			s := make([]*cluster.Cluster, 0, 16)
			return &s
		},
	}

	// HTTPFilterSlicePool предоставляет пул для слайсов HTTP фильтров
	HTTPFilterSlicePool = sync.Pool{
		New: func() interface{} {
			s := make([]*hcmv3.HttpFilter, 0, 8)
			return &s
		},
	}
)

// GetClusterSlice получает слайс кластеров из пула
func GetClusterSlice() *[]*cluster.Cluster {
	// Record object pool get operation
	RecordObjectPoolGet("cluster_slice")
	return ClusterSlicePool.Get().(*[]*cluster.Cluster)
}

// PutClusterSlice возвращает слайс кластеров в пул
// Важно: слайс должен быть освобожден от ссылок перед возвратом в пул
func PutClusterSlice(s *[]*cluster.Cluster) {
	if s == nil {
		return
	}
	*s = (*s)[:0] // Очистка слайса без освобождения памяти
	ClusterSlicePool.Put(s)
	// Record object pool put operation
	RecordObjectPoolPut("cluster_slice")
}

// GetHTTPFilterSlice получает слайс HTTP фильтров из пула
func GetHTTPFilterSlice() *[]*hcmv3.HttpFilter {
	// Record object pool get operation
	RecordObjectPoolGet("http_filter_slice")
	return HTTPFilterSlicePool.Get().(*[]*hcmv3.HttpFilter)
}

// PutHTTPFilterSlice возвращает слайс HTTP фильтров в пул
// Важно: слайс должен быть освобожден от ссылок перед возвратом в пул
func PutHTTPFilterSlice(s *[]*hcmv3.HttpFilter) {
	if s == nil {
		return
	}
	*s = (*s)[:0] // Очистка слайса без освобождения памяти
	HTTPFilterSlicePool.Put(s)
	// Record object pool put operation
	RecordObjectPoolPut("http_filter_slice")
}
