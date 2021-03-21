package pilotdiscovery

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/Stoakes/go-pkg/log"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
)

/* Public interface  of pilotdiscovery pkg */

const (
	// TypePrefix is the grpc type prefix
	TypePrefix = "type.googleapis.com/envoy.api.v2."

	/* XDS Constants */

	// ClusterType is used for cluster discovery. Typically first request received
	ClusterType = TypePrefix + "Cluster"
	// EndpointType is used for EDS and ADS endpoint discovery. Typically second request.
	EndpointType = TypePrefix + "ClusterLoadAssignment"
	// ListenerType is sent after clusters and endpoints.
	ListenerType = TypePrefix + "Listener"
	// RouteType is sent after listeners.
	RouteType = TypePrefix + "RouteConfiguration"
)

// PilotClient is a client to a Pilot Service Discovery Server
type PilotClient interface {
	AddWatch(host string, namespace string, port string) (chan EndpointsState, error)
	Shutdown()
	RefreshFromRemote() error
	EdszHandler(w http.ResponseWriter, req *http.Request)
	IsConnected() (bool, uint32)
}

// PilotClientOptions is a struct gathering every option required to start a PilotClient
type PilotClientOptions struct {
	// PilotURL is a valid <hostname>:<port> connection string
	PilotURL string
	// IP is the IP address of the application running this library
	IP string
	// AppName is the application name of the application running this library
	AppName string
	// Namespace is the Kubernetes namespace in which the application using this library is running
	Namespace string
	// DialOpts are the grpc DialOptions to dial Pilot Server
	DialOptions []grpc.DialOption
	// Backoff mechanism is copied from gRPC
	// Initial retry is at random(0, initialBackoffMS).
	// In general, the nth attempt will occur at random(0, min(InitialBackoff*BackoffMultiplier**(n-1), MaxBackoff)).
	// InitialBackOff is the duration. Default to 1 second
	InitialBackoff time.Duration
	// MaxBackOff is the maximum duration between 2 connection attempts in case of Pilot disconnection. Default to 2 minutes
	MaxBackoff time.Duration
	// BackoffMultiplier is the progression multiplier between initial and max backoff. Default to 1.6
	BackoffMultiplier float64
}

// pilotClient is the implementation of the PilotClient interface
type pilotClient struct {
	// connected is a boolean used to store connection to pilot existence
	connected *atomic.Bool
	// options Pilot Options, provided on start
	options PilotClientOptions
	// subscriptions struct storing internal subscriptions and endpoints knowledge
	subscriptions *stateStore
	// pilotConn TCP connection to Pilot server
	pilotConn *grpc.ClientConn
	// pilotStream gRPC stream to Pilot server
	pilotStream discoveryv2.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	// shutdown is Server shutting down ?
	shutdown *atomic.Bool
	// retryCounter How many times has the client tried to reconnect to Pilot since it's been disconnected.
	// Used for backoff delay computation. A reconnection sets it back to 0.
	retryCounter *atomic.Uint32
}

// NewPilotClient starts a subscription (gRPC stream) to pilot. When using several gRPC connections,
// we do not want to open multiple subscriptions to pilot.
// Therefore, we open only one connection to pilot and use channels for intra-app communication
func NewPilotClient(ctx context.Context, options PilotClientOptions) (PilotClient, error) {
	pc, err := newPilotClient(ctx, options)
	return pc, err
}

func newPilotClient(ctx context.Context, options PilotClientOptions) (*pilotClient, error) {
	// Options validation
	options, err := options.Validate()
	if err != nil {
		return &pilotClient{}, err
	}
	pClient := &pilotClient{
		shutdown:      atomic.NewBool(false),
		options:       options,
		subscriptions: newStateStore(),
		connected:     atomic.NewBool(false),
		retryCounter:  atomic.NewUint32(0),
	}
	err = pClient.connect(ctx)
	if err != nil {
		return &pilotClient{}, err
	}
	go func(ctx context.Context) {
		for {
			endpoints, err := pClient.pilotStream.Recv()
			if err != nil {
				if err == io.EOF {
					log.Bg().Error("EOF for Pilot stream")
				}
				if pClient.shutdown.Load() { // if client is shutting down, exit on error
					break
				}
				if err != nil {
					log.Bg().Error("Connection to Pilot closed with error: " + err.Error() + ". Trying to reconnect")
					// manual retry. Any contribution welcomed if better solution than this one.
					pClient.reconnect(ctx, err)
				}
				continue
			}
			pClient.handleRecv(endpoints)
		}
	}(ctx)
	return pClient, nil
}

// AddWatch takes an address, updates the subscription query and return a subscription channel where
// updates on the address will be sent. Get more details about EndpointsState in its documentation
func (p *pilotClient) AddWatch(host string, namespace string, port string) (chan EndpointsState, error) {
	req := &xdsapi.DiscoveryRequest{
		Node:          p.node(),
		TypeUrl:       configTypeToTypeURL("eds"),
		ResourceNames: append(p.subscriptions.watchedList(), p.subscriptions.format(host, namespace, port)),
	}
	err := p.pilotStream.Send(req)
	if err != nil {
		return nil, err
	}
	channel, err := p.subscriptions.addWatcher(host, namespace, port)
	if err != nil {
		return nil, err
	}
	log.Bg().Debugf("Add watch on %s.%s:%s", host, namespace, port)
	return channel, nil
}

// Shutdown stops the connection to Pilot and close every internal subscription channel
func (p *pilotClient) Shutdown() {
	p.shutdown.Store(true)
	if p.pilotConn != nil {
		_ = p.pilotConn.Close()
		p.pilotConn = nil
	}
	p.subscriptions.clear()
	_ = p.pilotStream.CloseSend()
}

// RefreshFromRemote issue a request to pilot to get update for every internal subscription
func (p *pilotClient) RefreshFromRemote() error {
	req := &xdsapi.DiscoveryRequest{
		Node:          p.node(),
		TypeUrl:       configTypeToTypeURL("eds"),
		ResourceNames: p.subscriptions.watchedList(),
	}
	log.Bg().Debugf("Sending discovery request for %v", p.subscriptions.watchedList())
	return p.pilotStream.Send(req)
}

// IsConnected returns if the client is connected to a Pilot instance and the number of retries.
// Number of retries is returned to help you create readiness strategies based on retry number (beware of cascading failures though).
// Warning: The retry counter is incremented after the first retry. Therefore, there is a small risk the result is not up to date.
func (p *pilotClient) IsConnected() (bool, uint32) {
	return p.connected.Load(), p.retryCounter.Load()
}

// node creates a pilot node object with node identity
func (p *pilotClient) node() *core1.Node {
	return &core1.Node{
		Id: makeNodeID(p.options.IP, p.options.AppName, p.options.Namespace),
	}
}

// edszHandler is an http handler to expose the state of currently known endpoints
func (p *pilotClient) EdszHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	p.subscriptions.Lock()
	defer p.subscriptions.Unlock()

	comma := false
	if len(p.subscriptions.edsMap) > 0 {
		fmt.Fprintln(w, "[")
		for _, eds := range p.subscriptions.edsMap {
			if comma {
				fmt.Fprint(w, ",\n")
			} else {
				comma = true
			}
			jsonm := &jsonpb.Marshaler{Indent: "  "}
			dbgString, _ := jsonm.MarshalToString(eds)
			if _, err := w.Write([]byte(dbgString)); err != nil {
				return
			}
		}
		fmt.Fprintln(w, "]")
		return
	}
	fmt.Fprintln(w, "[]")
}

// connect initialize connection to pilot.
// It is used on start and in case of connection failure
func (p *pilotClient) connect(ctx context.Context) error {
	log.For(ctx).Debug("Dialing Pilot server at " + p.options.PilotURL)
	var err error
	p.pilotConn, err = grpc.Dial(p.options.PilotURL, p.options.DialOptions...)
	if err != nil {
		return err
	}
	pService := discoveryv2.NewAggregatedDiscoveryServiceClient(p.pilotConn)
	p.pilotStream, err = pService.StreamAggregatedResources(ctx)
	if err != nil {
		return err
	}
	p.connected.Store(true)
	log.For(ctx).Debug("Stream to Pilot opened")
	return nil
}

// reconnect cleans current connection and try to re-open new ones + refresh state if it succeeds
func (p *pilotClient) reconnect(ctx context.Context, err error) {
	p.connected.Store(false)
	connectionErr := err
	for connectionErr != nil {
		// pilotClient.pilotStream = nil
		if p.pilotConn != nil { // If connection is still opened, closing it
			_ = p.pilotConn.Close()
			p.pilotConn = nil
		}
		connectionErr = p.connect(ctx) // connection attempt
		if connectionErr != nil {
			log.For(ctx).Error("Trying to reconnect to Pilot failed: " + connectionErr.Error())
			time.Sleep(p.backoff(p.retryCounter.Load()))
			p.retryCounter.Inc()
			continue
		}
		log.For(ctx).Infof("Reconnection succeeded after %d attempts", p.retryCounter.Load())
		p.retryCounter.Store(0)
		p.connected.Store(true)
		err := p.RefreshFromRemote()
		if err != nil {
			log.For(ctx).Error("Error refreshing from remote: " + err.Error())
		}
	}
}

func (p *pilotClient) backoff(retries uint32) time.Duration {
	if retries == 0 {
		return p.options.InitialBackoff
	}
	backoff := math.Min(float64(p.options.MaxBackoff),
		float64(p.options.InitialBackoff)*math.Pow(p.options.BackoffMultiplier, float64(retries)))
	// Randomize backoff delays so that if a cluster of requests start at
	// the same time, they won't operate in lockstep.
	backoff *= 1 + 0.2*(rand.Float64()*2-1)
	if backoff < 0 {
		return 0
	}
	return time.Duration(backoff)
}

// ack acknowledge a message to pilot
func (p *pilotClient) ack(msg *xdsapi.DiscoveryResponse) {
	_ = p.pilotStream.Send(&xdsapi.DiscoveryRequest{
		ResponseNonce: msg.Nonce,
		TypeUrl:       msg.TypeUrl,
		Node:          p.node(),
		VersionInfo:   msg.VersionInfo,
	})
}

// handleRecv handles incoming message from pilot stream and broadcast endpoints to internal subscribers.
// Especially it ignore every non endpoints that would be in the response
func (p *pilotClient) handleRecv(msg *xdsapi.DiscoveryResponse) {
	if msg == nil {
		log.Bg().Debug("Got nil discovery response from Pilot")
		return
	}
	eds := []*xdsapi.ClusterLoadAssignment{}
	for _, rsc := range msg.Resources { // Any
		valBytes := rsc.Value
		if rsc.TypeUrl == EndpointType {
			ll := &xdsapi.ClusterLoadAssignment{}
			_ = proto.Unmarshal(valBytes, ll)
			eds = append(eds, ll)
		}
	}

	p.subscriptions.version.Inc()
	p.ack(msg)
	if len(eds) > 0 {
		p.subscriptions.Lock()
		defer p.subscriptions.Unlock()
		for _, e := range eds {
			if _, ok := p.subscriptions.watchers[e.GetClusterName()]; !ok { // no watchers on this cluster name, skip
				continue
			}
			p.subscriptions.edsMap[e.GetClusterName()] = e
			i := 0
			ips := EndpointsState{}
			for _, endpoints := range e.Endpoints {
				for _, lbEndpoints := range endpoints.LbEndpoints {
					ips.Endpoints = append(ips.Endpoints, Endpoint{
						IP:     lbEndpoints.GetEndpoint().GetAddress().GetSocketAddress().GetAddress(),
						Region: endpoints.Locality.Region,
						Zone:   endpoints.Locality.Zone})
					i++
				}
			}
			log.Bg().Debugf("Got update for %s: %v", e.GetClusterName(), ips)
			j := 0
			for _, c := range p.subscriptions.watchers[e.GetClusterName()] { //notify channels
				select {
				case c <- ips:
				default:
					// will be executed when no receiver to get ips list message
				}
				j++
			}
		}
	}
}

// Validate set defaults & check constraints on PilotClientOptions
// in case of nil error, the option struct is ready to be safely used by a Pilot client
func (o PilotClientOptions) Validate() (PilotClientOptions, error) {
	var (
		pilotURLMinLength        = 3
		ipMinLength              = 8
		appNameMinLength         = 1
		namespaceMinLength       = 1
		initialBackoffDefault    = 1 * time.Second
		maxbackoffDefault        = 2 * time.Minute
		backoffMultiplierMin     = float64(1)
		backoffMultiplierDefault = float64(1.6)
	)
	if len(o.PilotURL) < pilotURLMinLength {
		return PilotClientOptions{}, fmt.Errorf("%s is not recognized as a valid PilotURL. Must be at least %d characters", o.PilotURL, pilotURLMinLength)
	}
	if len(o.IP) < ipMinLength {
		return PilotClientOptions{}, fmt.Errorf("%s is not recognized as a valid IP. Must be at least %d characters", o.IP, ipMinLength)
	}
	if len(o.AppName) < appNameMinLength {
		return PilotClientOptions{}, fmt.Errorf("%s is not recognized as a valid AppName. Must be at least %d character", o.AppName, appNameMinLength)
	}
	if len(o.Namespace) < namespaceMinLength {
		return PilotClientOptions{}, fmt.Errorf("%s is not recognized as a valid Namespace. Must be at least %d character", o.Namespace, namespaceMinLength)
	}
	if float64(o.InitialBackoff) > float64(o.MaxBackoff) {
		return PilotClientOptions{}, fmt.Errorf("Initial backoff %s is greater than max backoff %s. Must be lower or equal", o.InitialBackoff, o.MaxBackoff)
	}
	if o.BackoffMultiplier == 0 {
		o.BackoffMultiplier = backoffMultiplierDefault
	} else if o.BackoffMultiplier < backoffMultiplierMin {
		return PilotClientOptions{}, fmt.Errorf("%f is not valid backoff multiplier. Must be greater or equal to %f", o.BackoffMultiplier, backoffMultiplierMin)
	}
	/* Set defaults */
	if o.InitialBackoff == 0 {
		o.InitialBackoff = initialBackoffDefault
	}
	if o.MaxBackoff == 0 {
		o.MaxBackoff = maxbackoffDefault
	}

	if len(o.DialOptions) == 0 {
		o.DialOptions = []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.WithTimeout(10 * time.Second),
		}
	}
	return o, nil
}

// makeNodeID formats for istio pilot application identity. This name will determine which
// configuration the application will receive.
func makeNodeID(IP, Name, Namespace string) string {
	return fmt.Sprintf("sidecar~%s~%s.%s~%s.svc.cluster.local", IP, Name, Namespace, Namespace)
}

func configTypeToTypeURL(configType string) string {
	switch configType {
	case "lds":
		return ListenerType
	case "cds":
		return ClusterType
	case "rds":
		return RouteType
	case "eds":
		return EndpointType
	default:
		panic(fmt.Sprintf("Unknown type %s", configType))
	}
}
