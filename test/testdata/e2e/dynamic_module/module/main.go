// Package main is a minimal Envoy dynamic module for the e2e tests:
//   - the HTTP filter set_response_header sets a response header to the value
//     passed in filter_config;
//   - the listener filter count_connections counts accepted connections in the
//     counter named in filter_config.
//
// The Go SDK passes Go pointers to Envoy, so Envoy must run with
// GODEBUG=cgocheck=0, as in the upstream dynamic module examples.
package main

import (
	"errors"
	"fmt"

	sdk "github.com/envoyproxy/envoy/source/extensions/dynamic_modules/sdk/go"
	_ "github.com/envoyproxy/envoy/source/extensions/dynamic_modules/sdk/go/abi"
	"github.com/envoyproxy/envoy/source/extensions/dynamic_modules/sdk/go/shared"
)

const responseHeader = "x-e2e-dynamic-module"

func main() {}

func init() {
	sdk.RegisterHttpFilterConfigFactories(map[string]shared.HttpFilterConfigFactory{
		"set_response_header": &configFactory{},
	})
	sdk.RegisterListenerFilterConfigFactories(map[string]shared.ListenerFilterConfigFactory{
		"count_connections": &connCounterConfigFactory{},
	})
}

type configFactory struct {
	shared.EmptyHttpFilterConfigFactory
}

func (f *configFactory) Create(_ shared.HttpFilterConfigHandle, unparsedConfig []byte) (shared.HttpFilterFactory, error) {
	return &filterFactory{value: string(unparsedConfig)}, nil
}

type filterFactory struct {
	shared.EmptyHttpFilterFactory
	value string
}

func (f *filterFactory) Create(_ shared.HttpFilterHandle) shared.HttpFilter {
	return &filter{value: f.value}
}

type filter struct {
	shared.EmptyHttpFilter
	value string
}

func (f *filter) OnResponseHeaders(headers shared.HeaderMap, _ bool) shared.HeadersStatus {
	headers.Set(responseHeader, f.value)
	return shared.HeadersStatusContinue
}

type connCounterConfigFactory struct {
	shared.EmptyListenerFilterConfigFactory
}

func (f *connCounterConfigFactory) Create(handle shared.ListenerFilterConfigHandle, unparsedConfig []byte) (shared.ListenerFilterFactory, error) {
	name := string(unparsedConfig)
	if name == "" {
		return nil, errors.New("filter_config must hold the counter name")
	}
	counter, result := handle.DefineCounter(name)
	if result != shared.MetricsSuccess {
		return nil, fmt.Errorf("failed to define counter %s: %d", name, result)
	}
	return &connCounterFactory{counter: counter}, nil
}

type connCounterFactory struct {
	shared.EmptyListenerFilterFactory
	counter shared.MetricID
}

func (f *connCounterFactory) Create(handle shared.ListenerFilterHandle) shared.ListenerFilter {
	return &connCounter{handle: handle, counter: f.counter}
}

type connCounter struct {
	shared.EmptyListenerFilter
	handle  shared.ListenerFilterHandle
	counter shared.MetricID
}

func (f *connCounter) OnAccept() shared.ListenerFilterStatus {
	f.handle.IncrementCounterValue(f.counter, 1)
	return shared.ListenerFilterStatusContinue
}
