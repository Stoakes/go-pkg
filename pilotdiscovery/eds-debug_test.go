package pilotdiscovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

// testMakePilotClient creates a new pilotClient without starting a subscription to Pilot
func testMakePilotClient() pilotClient {
	o, _ := PilotClientOptions{
		PilotURL:  "istio-pilot.istio-system",
		IP:        "192.168.0.100",
		AppName:   "testApp",
		Namespace: "istio-system",
	}.Validate()
	return pilotClient{
		options:       o,
		subscriptions: newStateStore(),
	}
}

// TestEDSOutputNoEndpoins tests the returned the JSON matches when no endpoint is known
// when no endpoints, we expect a 200 code and []
func TestEDSOutputNoEndpoints(t *testing.T) {
	pClient := testMakePilotClient() // assign pkg variable

	router := http.NewServeMux()
	router.HandleFunc("/debug/edsz", pClient.EdszHandler)
	server := httptest.NewServer(router)
	defer server.Close()

	// Build request
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/debug/edsz", server.URL), nil)
	if err != nil {
		t.Errorf("Error creating http request %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Error doing http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid response status code: %v", resp.Status)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading request body %v", err)
	}
	edsList := []string{} // no need for a sophisticated type, array must be epty
	if err := json.Unmarshal(bodyBytes, &edsList); err != nil {
		t.Errorf("Error unmarshalling body: %v", err)
	}
	if len(edsList) != 0 {
		t.Errorf("invalid response body. Expecting empty array got %s", string(bodyBytes))
	}
}

// TestEDSOutputWithEndpoins tests the returned the JSON matches the known state when some endpoints are known
// when some endpoints, we expect a list of endpoints and their localized ips.
func TestEDSOutputWithEndpoints(t *testing.T) {
	pClient := testMakePilotClient() // assign pkg variable
	for _, eds := range testCreateEdsMap(t, 2) {
		pClient.subscriptions.edsMap[eds.GetClusterName()] = eds
	}

	router := http.NewServeMux()
	router.HandleFunc("/debug/edsz", pClient.EdszHandler)
	server := httptest.NewServer(router)
	defer server.Close()

	// Build request
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/debug/edsz", server.URL), nil)
	if err != nil {
		t.Errorf("Error creating http request %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Error doing http request %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid response status code: %v", resp.Status)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading request body %v", err)
	}

	if !strings.Contains(string(bodyBytes), "outbound|6565||hello-service.default.svc.cluster.local") {
		t.Error("Response body does not contains outbound|6565||hello-service.default.svc.cluster.local", string(bodyBytes))
	}

	if !strings.Contains(string(bodyBytes), "outbound|6565||bonjour-service.default.svc.cluster.local") {
		t.Error("Response body does not contains outbound|6565||bonjour-service.default.svc.cluster.local", string(bodyBytes))
	}

	if !strings.Contains(string(bodyBytes), "10.4.0.16") {
		t.Error("Response body does not contains 10.4.0.16", string(bodyBytes))
	}

	if !strings.Contains(string(bodyBytes), "10.4.1.10") {
		t.Error("Response body does not contains 10.4.1.10", string(bodyBytes))
	}

}

func testCreateEdsMap(t *testing.T, length int) []*xdsapi.ClusterLoadAssignment {

	helloCLA := xdsapi.ClusterLoadAssignment{
		ClusterName: "outbound|6565||hello-service.default.svc.cluster.local",
		Endpoints: []*endpoint.LocalityLbEndpoints{
			&endpoint.LocalityLbEndpoints{
				LoadBalancingWeight: &wrappers.UInt32Value{
					Value: uint32(2),
				},
				Locality: &core.Locality{
					Region: "europe-west1",
					Zone:   "europe-west1-b",
				},
				LbEndpoints: []*endpoint.LbEndpoint{
					&endpoint.LbEndpoint{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Address: "10.4.0.16",
											PortSpecifier: &core.SocketAddress_PortValue{
												PortValue: uint32(6565),
											},
										},
									},
								},
							},
						},
					},
					&endpoint.LbEndpoint{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Address: "10.4.1.18",
											PortSpecifier: &core.SocketAddress_PortValue{
												PortValue: uint32(6565),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			&endpoint.LocalityLbEndpoints{
				LoadBalancingWeight: &wrappers.UInt32Value{
					Value: uint32(1),
				},
				Locality: &core.Locality{
					Region: "europe-west1",
					Zone:   "europe-west1-d",
				},
				LbEndpoints: []*endpoint.LbEndpoint{
					&endpoint.LbEndpoint{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Address: "10.4.2.2",
											PortSpecifier: &core.SocketAddress_PortValue{
												PortValue: uint32(6565),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bonjourCLA := xdsapi.ClusterLoadAssignment{
		ClusterName: "outbound|6565||bonjour-service.default.svc.cluster.local",
		Endpoints: []*endpoint.LocalityLbEndpoints{
			&endpoint.LocalityLbEndpoints{
				LoadBalancingWeight: &wrappers.UInt32Value{
					Value: uint32(1),
				},
				Locality: &core.Locality{
					Region: "europe-west1",
					Zone:   "europe-west1-b",
				},
				LbEndpoints: []*endpoint.LbEndpoint{
					&endpoint.LbEndpoint{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Address: "10.4.1.10",
											PortSpecifier: &core.SocketAddress_PortValue{
												PortValue: uint32(6565),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if length == 1 {
		return []*xdsapi.ClusterLoadAssignment{&helloCLA}
	} else if length == 2 {
		return []*xdsapi.ClusterLoadAssignment{&helloCLA, &bonjourCLA}
	}
	t.Fatal("Invalid length value. expecting 1 or 2")
	return nil
}
