package pilotdiscovery

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	pilotapi "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
)

// adsServerMock is a small gRPC server mocking pilot behaviour for tests
// Every opened stream is available to push objects on it and simulate a pilot response/push
type adsServerMock struct {
	grpcServer        *grpc.Server
	address           string
	streams           map[uint64]*streamStore
	connectionCounter *atomic.Uint64
}

// streamStore is an open connection and its control channel
type streamStore struct {
	stream  pilotapi.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	id      uint64
	control chan *xdsapi.DiscoveryResponse // every response posted on this channel will be sent on the stream
	quit    chan bool                      // a message on this channel will terminate the stream
}

// newADSServerMock creates a mock ADS server and starts it on an available port
// returns are: the server Mock, the funtion to shut it down, the address to connect to the server
func newADSServerMock(t *testing.T, port string) *adsServerMock {
	if len(port) == 0 {
		port = ":0"
	}
	listener, err := net.Listen("tcp", port)
	if err != nil {
		t.Fatal(err)
	}
	adsServer := &adsServerMock{
		grpcServer:        grpc.NewServer(),
		streams:           make(map[uint64]*streamStore),
		connectionCounter: atomic.NewUint64(0),
	}
	pilotapi.RegisterAggregatedDiscoveryServiceServer(adsServer.grpcServer, adsServer)
	go func() {
		err := adsServer.grpcServer.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			t.Error("Error opening grpc server:", err.Error())
		}
	}()
	adsServer.address = "localhost:" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	return adsServer
}

// StreamAggregatedResources implements ADS Server interface
func (a *adsServerMock) StreamAggregatedResources(stream pilotapi.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	ss := &streamStore{
		stream:  stream,
		id:      a.connectionCounter.Load(),
		control: make(chan *xdsapi.DiscoveryResponse),
		quit:    make(chan bool),
	}
	a.connectionCounter.Inc()
	a.streams[ss.id] = ss
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			delete(a.streams, ss.id)
			return nil
		case <-ss.quit:
			return nil
		case xds := <-ss.control:
			stream.Send(xds)
		}
	}
}

// DeltaAggregatedResources implements ADS Server interface
func (a *adsServerMock) DeltaAggregatedResources(stream pilotapi.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return fmt.Errorf("Unimplemented")
}

// Shutdown shuts down the server mock
func (a *adsServerMock) Shutdown() {
	time.Sleep(1 * time.Millisecond)
	for _, stream := range a.streams {
		stream.quit <- true
	}
	a.grpcServer.Stop()
	time.Sleep(3 * time.Millisecond)
}

func (a *adsServerMock) Address() string {
	return a.address
}

func endpointDiscoveryResponse(loadAssignments []*xdsapi.ClusterLoadAssignment, version string) *xdsapi.DiscoveryResponse {
	out := &xdsapi.DiscoveryResponse{
		TypeUrl: EndpointType,
		// Pilot does not really care for versioning. It always supplies what's currently
		// available to it, irrespective of whether Envoy chooses to accept or reject EDS
		// responses. Pilot believes in eventual consistency and that at some point, Envoy
		// will begin seeing results it deems to be good.
		VersionInfo: version,
		Nonce:       time.Now().String(),
	}
	for _, loadAssignment := range loadAssignments {
		resource, _ := messageToAny(loadAssignment)
		out.Resources = append(out.Resources, resource)
	}

	return out
}

func messageToAny(msg proto.Message) (*any.Any, error) {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &any.Any{
		TypeUrl: "type.googleapis.com/" + proto.MessageName(msg),
		Value:   b.Bytes(),
	}, nil
}
