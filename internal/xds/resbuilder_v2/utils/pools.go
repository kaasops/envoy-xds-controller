package utils

import (
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

var (
	// StringSlicePool предоставляет пул для слайсов строк (имена кластеров, домены и т.д.)
	StringSlicePool = sync.Pool{
		New: func() interface{} {
			s := make([]string, 0, 8)
			return &s
		},
	}

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

// GetStringSlice получает слайс строк из пула
func GetStringSlice() *[]string {
	return StringSlicePool.Get().(*[]string)
}

// PutStringSlice возвращает слайс строк в пул
// Важно: слайс должен быть освобожден от ссылок перед возвратом в пул
func PutStringSlice(s *[]string) {
	if s == nil {
		return
	}
	*s = (*s)[:0] // Очистка слайса без освобождения памяти
	StringSlicePool.Put(s)
}

// GetClusterSlice получает слайс кластеров из пула
func GetClusterSlice() *[]*cluster.Cluster {
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
}

// GetHTTPFilterSlice получает слайс HTTP фильтров из пула
func GetHTTPFilterSlice() *[]*hcmv3.HttpFilter {
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
}

// SafeCopyStringSlice создает копию слайса строк с использованием пула
// Возвращает новый слайс, который нужно освободить с помощью PutStringSlice
func SafeCopyStringSlice(src []string) *[]string {
	if len(src) == 0 {
		return GetStringSlice()
	}

	dst := GetStringSlice()
	*dst = append(*dst, src...)
	return dst
}

// SafeCopyClusterSlice создает глубокую копию слайса кластеров с использованием пула
// Возвращает новый слайс, который нужно освободить с помощью PutClusterSlice
func SafeCopyClusterSlice(src []*cluster.Cluster) *[]*cluster.Cluster {
	if len(src) == 0 {
		return GetClusterSlice()
	}

	dst := GetClusterSlice()
	for _, c := range src {
		// Пустые указатели не копируем
		if c == nil {
			continue
		}
		*dst = append(*dst, c)
	}
	return dst
}

// SafeCopyHTTPFilterSlice создает глубокую копию слайса HTTP фильтров с использованием пула
// Возвращает новый слайс, который нужно освободить с помощью PutHTTPFilterSlice
func SafeCopyHTTPFilterSlice(src []*hcmv3.HttpFilter) *[]*hcmv3.HttpFilter {
	if len(src) == 0 {
		return GetHTTPFilterSlice()
	}

	dst := GetHTTPFilterSlice()
	for _, f := range src {
		// Пустые указатели не копируем
		if f == nil {
			continue
		}
		*dst = append(*dst, f)
	}
	return dst
}
