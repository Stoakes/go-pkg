package pilotdiscovery

import (
	"testing"
)

// TeststateStore tests the subscription manager (the component storing internal subscriptions and known endpoints)
// works as expected
func TestStateStore(t *testing.T) {
	subManager := newStateStore()

	// We can't test if a lock is locked or not. So these 2 are mostly to improve coverage.
	subManager.Lock()
	subManager.Unlock()

	/* format */
	if subManager.format("host", "ns", "6532") != "outbound|6532||host.ns.svc.cluster.local" {
		t.Fatal("(host, namespace, port) tuple formatting failed, got ", subManager.format("host", "ns", "6532"))
	}
	/* addWatch */
	// add one subscription to one new name
	_, err := subManager.addWatcher("morgen", "germany", "6565")
	if err != nil {
		t.Fatal("Adding watch on morgen.germany:6565 failed: ", err.Error())
	}
	if _, ok := subManager.watchers["outbound|6565||morgen.germany.svc.cluster.local"]; !ok {
		t.Fatal("Add watchers does not update stateStore watchers map")
	}
	if len(subManager.watchers) != 1 {
		t.Fatal("There must be only one item in watchers map, found", len(subManager.watchers))
	}
	if len(subManager.watchers["outbound|6565||morgen.germany.svc.cluster.local"]) != 1 {
		t.Fatal("There must be 1 channel in watchers[morgen.germany:6565] list, found", len(subManager.watchers["outbound|6565||morgen.germany.svc.cluster.local"]))
	}
	// add one subscription to another new name
	_, err = subManager.addWatcher("ola", "spain", "1212")
	if err != nil {
		t.Fatal("Adding watch on ola.spain:1212 failed: ", err.Error())
	}
	if _, ok := subManager.watchers["outbound|1212||ola.spain.svc.cluster.local"]; !ok {
		t.Fatal("Add watchers does not update stateStore watchers map 2")
	}
	if len(subManager.watchers) != 2 {
		t.Fatal("There must be 2 items in watchers map, found", len(subManager.watchers))
	}
	if len(subManager.watchers["outbound|1212||ola.spain.svc.cluster.local"]) != 1 {
		t.Fatal("There must be 1 channel in watchers[ola.spain:1212] list, found", len(subManager.watchers["outbound|1212||ola.spain.svc.cluster.local"]))
	}
	// add one new subscription to an already existing subscription
	_, err = subManager.addWatcher("morgen", "germany", "6565")
	if err != nil {
		t.Fatal("Adding watch on morgen.germany:6565 failed: ", err.Error())
	}
	if _, ok := subManager.watchers["outbound|6565||morgen.germany.svc.cluster.local"]; !ok {
		t.Fatal("Add watchers does not update stateStore watchers map 3")
	}
	if len(subManager.watchers) != 2 {
		t.Fatal("There must be 2 items in watchers map, found", len(subManager.watchers))
	}
	if len(subManager.watchers["outbound|6565||morgen.germany.svc.cluster.local"]) != 2 {
		t.Fatal("There must be 2 channels in watchers[morgen.germany:6565] list, found", len(subManager.watchers["outbound|6565||morgen.germany.svc.cluster.local"]))
	}

	/* watchedList */
	if res := subManager.watchedList(); (res[0] != "outbound|6565||morgen.germany.svc.cluster.local" && res[1] != "outbound|6565||morgen.germany.svc.cluster.local") ||
		(res[0] != "outbound|1212||ola.spain.svc.cluster.local" && res[1] != "outbound|1212||ola.spain.svc.cluster.local") ||
		len(res) != 2 {
		t.Fatal("watchedList does not return an accurate representation of key in subManager[watchers]: ", res)
	}
}

// TestConfigTypeToTypeURL test shortname for envoy types are correctly mapped to their fully qualified name
func TestConfigTypeToTypeURL(t *testing.T) {
	if configTypeToTypeURL("lds") != "type.googleapis.com/envoy.api.v2.Listener" {
		t.Error("configTypeToTypeURL(lds) does not produce expected output")
	}
	if configTypeToTypeURL("cds") != "type.googleapis.com/envoy.api.v2.Cluster" {
		t.Error("configTypeToTypeURL(cds) does not produce expected output")
	}
	if configTypeToTypeURL("rds") != "type.googleapis.com/envoy.api.v2.RouteConfiguration" {
		t.Error("configTypeToTypeURL(rds) does not produce expected output")
	}
	if configTypeToTypeURL("eds") != "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" {
		t.Error("configTypeToTypeURL(eds) does not produce expected output")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("configTypeToTypeURL(antoine) did not panic while it must")
		}
	}()
	configTypeToTypeURL("antoine")
}
