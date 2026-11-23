package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaasops/envoy-xds-controller/internal/filewatcher"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
	"go.uber.org/zap"
)

// The registry given to New is the one the running API lists, under the name
// of this host. The server started here lives until the test process exits.
func TestRunServesTheRegistryGivenToNew(t *testing.T) {
	watcher, err := filewatcher.NewFileWatcher()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(watcher.Cancel)
	registry := clients.NewRegistry()
	key := clients.Key{Protocol: clients.Delta, ID: 1}
	registry.Set(key, &clients.Info{Address: "10.0.0.1:41000"})
	registry.Update(key, &clients.Info{NodeID: "core", Version: "v1.40.2"})

	configPath := filepath.Join(t.TempDir(), "config.json")
	server, err := New(xdscache.NewSnapshotCache(), &Config{}, zap.NewNop(), false, watcher, configPath, registry)
	if err != nil {
		t.Fatal(err)
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	_ = lis.Close()
	go func() { _ = server.Run(port, "http", fmt.Sprintf("localhost:%d", port)) }()

	var body struct {
		ControllerPod string `json:"controllerPod"`
		Clients       []struct {
			NodeID string `json:"nodeId"`
		} `json:"clients"`
	}
	var status int
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/clients", port))
		if err == nil {
			status = resp.StatusCode
			_ = json.NewDecoder(resp.Body).Decode(&body)
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the API did not start: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	hostname, _ := os.Hostname()
	if status != http.StatusOK || len(body.Clients) != 1 || body.Clients[0].NodeID != "core" {
		t.Errorf("status %d, body %+v, want the stream of node core", status, body)
	}
	if hostname == "" || body.ControllerPod != hostname {
		t.Errorf("controllerPod %q, want the host name %q", body.ControllerPod, hostname)
	}
}
