package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Cache metrics

	// CacheHits tracks the number of successful cache lookups
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_cache_hits_total",
			Help: "Total number of cache hits in resbuilder",
		},
		[]string{"cache_type"},
	)

	// CacheMisses tracks the number of failed cache lookups
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_cache_misses_total",
			Help: "Total number of cache misses in resbuilder",
		},
		[]string{"cache_type"},
	)

	// CacheSize tracks the current number of items in caches
	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "envoy_xds_resbuilder_cache_size",
			Help: "Current number of items in resbuilder caches",
		},
		[]string{"cache_type"},
	)

	// CacheEvictions tracks the number of items evicted from caches
	CacheEvictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_cache_evictions_total",
			Help: "Total number of items evicted from resbuilder caches",
		},
		[]string{"cache_type", "reason"}, // reason can be "lru", "ttl", "manual", etc.
	)

	// CacheItemAge tracks the age of items in the cache at time of access
	CacheItemAge = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "envoy_xds_resbuilder_cache_item_age_seconds",
			Help:    "Age of items in the cache at time of access",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // From 1s to ~17m
		},
		[]string{"cache_type", "result"}, // result can be "hit" or "miss"
	)

	// Performance metrics

	// BuildDuration tracks the time spent on building resources
	BuildDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "envoy_xds_resbuilder_build_duration_seconds",
			Help:    "Time spent building resources in resbuilder",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"resource_type", "operation"},
	)

	// ComponentDuration tracks the time spent in specific components
	ComponentDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "envoy_xds_resbuilder_component_duration_seconds",
			Help:    "Time spent in specific components of the build process",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // From 0.1ms to ~0.4s
		},
		[]string{"component", "method", "status"}, // status can be "success" or "error"
	)

	// ResourceProcessingTime tracks the time spent processing different resource types
	ResourceProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "envoy_xds_resbuilder_resource_processing_seconds",
			Help:    "Time spent processing different resource types",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // From 0.1ms to ~0.4s
		},
		[]string{"resource_type", "operation", "implementation"}, // implementation can be "original" or "mainbuilder"
	)

	// Memory metrics

	// ObjectPoolGets tracks the number of objects retrieved from pools
	ObjectPoolGets = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_object_pool_gets_total",
			Help: "Total number of objects retrieved from pools",
		},
		[]string{"pool_type"},
	)

	// ObjectPoolPuts tracks the number of objects returned to pools
	ObjectPoolPuts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_object_pool_puts_total",
			Help: "Total number of objects returned to pools",
		},
		[]string{"pool_type"},
	)

	// ResourceCounts tracks the number of resources created
	ResourceCounts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_resources_created_total",
			Help: "Total number of resources created by resbuilder",
		},
		[]string{"resource_type", "implementation"}, // implementation can be "original" or "mainbuilder"
	)

	// MemoryUsage estimates memory usage for large operations
	MemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "envoy_xds_resbuilder_memory_usage_bytes",
			Help: "Estimated memory usage for large operations in bytes",
		},
		[]string{"operation", "implementation"}, // implementation can be "original" or "mainbuilder"
	)

	// ResourceCardinality tracks the number of items in various collections
	ResourceCardinality = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "envoy_xds_resbuilder_resource_cardinality",
			Help: "Number of items in various resource collections",
		},
		[]string{"collection_type", "resource_type", "implementation"}, // implementation can be "original" or "mainbuilder"
	)

	// FeatureFlagMetrics tracks feature flag usage
	FeatureFlagMetrics = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envoy_xds_resbuilder_feature_flag_usage_total",
			Help: "Usage counts for feature flags",
		},
		[]string{"flag_name", "value"},
	)
)

// RecordCacheHit increments the cache hit counter for the specified cache type
func RecordCacheHit(cacheType string) {
	CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss increments the cache miss counter for the specified cache type
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}

// UpdateCacheSize sets the current cache size for the specified cache type
func UpdateCacheSize(cacheType string, size int) {
	CacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// RecordCacheEviction increments the cache eviction counter for the specified cache type and reason
func RecordCacheEviction(cacheType, reason string) {
	CacheEvictions.WithLabelValues(cacheType, reason).Inc()
}

// RecordBuildDuration records the time spent building a resource of the specified type
func RecordBuildDuration(resourceType, operation string, durationSeconds float64) {
	BuildDuration.WithLabelValues(resourceType, operation).Observe(durationSeconds)
}

// RecordObjectPoolGet increments the object pool get counter for the specified pool type
func RecordObjectPoolGet(poolType string) {
	ObjectPoolGets.WithLabelValues(poolType).Inc()
}

// RecordObjectPoolPut increments the object pool put counter for the specified pool type
func RecordObjectPoolPut(poolType string) {
	ObjectPoolPuts.WithLabelValues(poolType).Inc()
}

// RecordResourceCreation increments the resource creation counter for the specified resource type
func RecordResourceCreation(resourceType string) {
	ResourceCounts.WithLabelValues(resourceType).Inc()
}
