package store

import "testing"

func TestGetNodeDomainsForNodes_BasicAndMissing(t *testing.T) {
	s := New()

	idx := map[string]map[string]struct{}{
		"nodeA": {},
		"nodeB": {"a.example": {}, "b.example": {}},
	}
	s.ReplaceNodeDomainsIndex(idx)

	res, missing := s.GetNodeDomainsForNodes([]string{"nodeA", "nodeB", "nodeC"})

	if len(missing) != 1 || missing[0] != "nodeC" {
		t.Fatalf("expected missing=[nodeC], got %v", missing)
	}
	if set, ok := res["nodeA"]; !ok || len(set) != 0 {
		t.Fatalf("expected empty set for nodeA, got %+v, ok=%v", set, ok)
	}
	if set, ok := res["nodeB"]; !ok || len(set) != 2 {
		t.Fatalf("expected 2 domains for nodeB, got %+v, ok=%v", set, ok)
	}
	if _, ok := res["nodeC"]; ok {
		t.Fatalf("did not expect entry for nodeC in result map")
	}
}

func TestGetNodeDomainsForNodes_DeepCopyImmutability(t *testing.T) {
	s := New()
	idx := map[string]map[string]struct{}{
		"n": {"d1": {}, "d2": {}},
	}
	s.ReplaceNodeDomainsIndex(idx)

	res1, _ := s.GetNodeDomainsForNodes([]string{"n"})
	res1["n"]["injected"] = struct{}{}

	res2, _ := s.GetNodeDomainsForNodes([]string{"n"})
	if len(res2["n"]) != 2 {
		t.Fatalf("expected original index to be unaffected (2 domains), got %d", len(res2["n"]))
	}
}
