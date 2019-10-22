package pilotdiscovery

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Stoakes/go-pkg/log"
	"google.golang.org/grpc"
)

// TestNewPilotClientNoGRPCOptions starts a new test pilot server and connect to it.
// small test to just get no grpc dial option tests coverage
func TestNewPilotClientDefaultOptions(t *testing.T) {
	adsMock := newADSServerMock(t, "")
	defer adsMock.Shutdown()
	pClient, err := newPilotClient(context.Background(), PilotClientOptions{
		PilotURL:  adsMock.Address(),
		IP:        "127.0.0.1",
		AppName:   "testAgent",
		Namespace: "testns",
	})
	if err != nil {
		t.Error("Cannot start new pilot client " + err.Error())
	}
	if pClient.options.InitialBackoff == 0 {
		t.Error("pilotClient.Options.InitialBackoff should not be 0")
	}
	if pClient.options.MaxBackoff == 0 {
		t.Error("pilotClient.Options.MaxBackoff should not be 0")
	}
	if pClient.options.BackoffMultiplier == 0 {
		t.Error("pilotClient.Options.BackoffMultiplier should not be 0")
	}
	if len(pClient.options.DialOptions) == 0 {
		t.Error("len(pilotClient.Options.DialOptions) should not be 0")
	}
	defer pClient.Shutdown()
	if pClient.pilotStream == nil {
		t.Errorf("PilotClient.PilotStream is empty while it should not be.")
	}
}

type invalidOptionsTestCase struct {
	options      PilotClientOptions
	errorMessage string
}

// TestNewPilotInvalidOptions test return when provided options are invalid
func TestNewPilotInvalidOptions(t *testing.T) {

	tt := map[int]invalidOptionsTestCase{
		1: invalidOptionsTestCase{
			options: PilotClientOptions{
				PilotURL:  "pilot.def:8063",
				IP:        "u",
				AppName:   "testAgent",
				Namespace: "testns",
			},
			errorMessage: "u is not recognized as a valid IP. Must be at least 8 characters",
		},
		2: invalidOptionsTestCase{
			options: PilotClientOptions{
				PilotURL:       "pilot.def:8063",
				IP:             "1.1.11.1",
				AppName:        "testAgent",
				Namespace:      "testns",
				InitialBackoff: time.Second,
				MaxBackoff:     time.Millisecond,
			},
			errorMessage: "Initial backoff 1s is greater than max backoff 1ms. Must be lower or equal",
		},
		3: invalidOptionsTestCase{
			options: PilotClientOptions{
				PilotURL:          "pilot.def:8063",
				IP:                "1.1.11.1",
				AppName:           "testAgent",
				Namespace:         "testns",
				InitialBackoff:    time.Millisecond,
				MaxBackoff:        time.Second,
				BackoffMultiplier: 0.6,
			},
			errorMessage: "0.600000 is not valid backoff multiplier. Must be greater or equal to 1.000000",
		},
		4: invalidOptionsTestCase{
			options: PilotClientOptions{
				PilotURL:  "e",
				IP:        "1.1.11.1",
				AppName:   "testAgent",
				Namespace: "testns",
			},
			errorMessage: "e is not recognized as a valid PilotURL. Must be at least 3 characters",
		},
		5: invalidOptionsTestCase{
			options: PilotClientOptions{
				PilotURL:  "pilot.default:8986",
				IP:        "1.1.11.1",
				Namespace: "testns",
			},
			errorMessage: " is not recognized as a valid AppName. Must be at least 1 character",
		},
		6: invalidOptionsTestCase{
			options: PilotClientOptions{
				PilotURL: "pilot.default:8986",
				IP:       "1.1.11.1",
				AppName:  "app",
			},
			errorMessage: " is not recognized as a valid Namespace. Must be at least 1 character",
		},
	}

	for k, v := range tt {
		_, err := NewPilotClient(context.Background(), v.options)
		if err == nil {
			t.Errorf("Test %d Invalid NewPilotClient options should return non empty error message", k)
		}
		if err.Error() != v.errorMessage {
			t.Errorf("Test %d. Expected error message %s Got %s", k, v.errorMessage, err.Error())
		}
	}
}

// TestNewPilotClient starts a new test pilot server and connect to it.
// it also add some watchers and send updates to check these are correctly propagated
func TestNewPilotClient(t *testing.T) {
	log.Setup(context.Background(), log.Options{LogLevel: "debug", Debug: true})
	adsMock := newADSServerMock(t, "")
	defer adsMock.Shutdown()
	pClient, err := newPilotClient(context.Background(), PilotClientOptions{
		PilotURL:    adsMock.Address(),
		IP:          "127.0.0.1",
		AppName:     "testAgent",
		Namespace:   "testns",
		DialOptions: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3 * time.Second)},
	})
	if err != nil {
		t.Fatal("Cannot start new pilot client " + err.Error())
	}
	defer pClient.Shutdown()
	if pClient.pilotStream == nil {
		t.Errorf("PilotClient.PilotStream is empty while it should not be. 2")
	}
	if len(pClient.subscriptions.watchers) != 0 {
		t.Error("subscription length should be 0")
	}
	helloChannel, err := pClient.AddWatch("hello-service", "default", "6565")
	helloChannelMsgCounter := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		select {
		case <-helloChannel:
			helloChannelMsgCounter++
		case <-ctx.Done():
			return
		}
	}(ctx)
	if len(pClient.subscriptions.watchedList()) != 1 {
		t.Error("Number of watchers should be equal to 1")
	}
	time.Sleep(10 * time.Millisecond) // giving ads server some time
	if len(adsMock.streams) != 1 {
		t.Fatal("There must be 1 connection open to the ADS server. Got", len(adsMock.streams))
	}
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 1), "1")
	time.Sleep(5 * time.Millisecond)
	if len(pClient.subscriptions.edsMap) != 1 {
		t.Error("EdsMap length should be equal to 1. Got", len(pClient.subscriptions.edsMap))
	}
	if helloChannelMsgCounter != 1 {
		t.Error("Channel subscribed to hello-service.default:6565 should have received 1 message. Got", helloChannelMsgCounter)
	}
}

// TestMultipleSubscriptions simulates several internal subscriptions and
func TestMultipleSubscriptions(t *testing.T) {
	adsMock := newADSServerMock(t, "")
	defer adsMock.Shutdown()
	pClient, err := newPilotClient(context.Background(), PilotClientOptions{
		PilotURL:    adsMock.Address(),
		IP:          "127.0.0.1",
		AppName:     "testAgent",
		Namespace:   "testns",
		DialOptions: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3 * time.Second)},
	})
	if err != nil {
		t.Fatal("Cannot start subscription " + err.Error())
	}
	defer pClient.Shutdown()

	helloChannel, err := pClient.AddWatch("hello-service", "default", "6565")
	helloChannelMsgCounter := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case <-helloChannel:
				helloChannelMsgCounter++
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	bonjourChannel, err := pClient.AddWatch("bonjour-service", "default", "6565")
	bonjourChannelMsgCounter := 0
	go func(ctx context.Context) {
		for {
			select {
			case <-bonjourChannel:
				bonjourChannelMsgCounter++
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	olaChannel, err := pClient.AddWatch("ola-service", "default", "6565")
	olaChannelMsgCounter := 0
	go func(ctx context.Context) {
		for {
			select {
			case <-olaChannel:
				olaChannelMsgCounter++
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	if len(pClient.subscriptions.watchedList()) != 3 {
		t.Error("Number of watchers should be equal to 3")
	}
	time.Sleep(5 * time.Millisecond) // giving ads server some time
	if len(adsMock.streams) != 1 {
		t.Fatal("There must be 1 connection open to the ADS server. Got", len(adsMock.streams))
	}
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 1), "1")
	time.Sleep(5 * time.Millisecond)
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 2), "2")
	time.Sleep(5 * time.Millisecond)
	if len(pClient.subscriptions.edsMap) != 2 {
		t.Error("EdsMap length should be equal to 2. Got", len(pClient.subscriptions.edsMap))
	}
	if helloChannelMsgCounter != 2 {
		t.Error("Channel subscribed to hello-service.default:6565 should have received 2 messages. Got", helloChannelMsgCounter)
	}
	if bonjourChannelMsgCounter != 1 {
		t.Error("Channel subscribed to bonjour-service.default:6565 should have received 1 message. Got", bonjourChannelMsgCounter)
	}
	if olaChannelMsgCounter != 0 {
		t.Error("Channel subscribed to la-service.default:6565 should have received 0 message. Got", olaChannelMsgCounter)
	}
}

// TestNoSubscriber simulates messages receptions when there is no internal subscriptions
// this should normally not happen because library subscribe only to endpoints when there's an internal consumer
// but we can imagine internal consumer to be closed and subscription to not be reovked.
func TestNoSubscriber(t *testing.T) {
	adsMock := newADSServerMock(t, "")
	defer adsMock.Shutdown()
	pClient, err := newPilotClient(context.Background(), PilotClientOptions{
		PilotURL:    adsMock.Address(),
		IP:          "127.0.0.1",
		AppName:     "testAgent",
		Namespace:   "testns",
		DialOptions: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3 * time.Second)},
	})
	if err != nil {
		t.Fatal("Cannot start pilot client " + err.Error())
	}
	defer pClient.Shutdown()
	if len(pClient.subscriptions.watchedList()) != 0 {
		t.Error("Number of watchers should be equal to 0")
	}
	time.Sleep(5 * time.Millisecond) // giving ads server some time
	if len(adsMock.streams) != 1 {
		t.Fatal("There must be 1 connection open to the ADS server. Got", len(adsMock.streams))
	}
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 1), "1")
	time.Sleep(5 * time.Millisecond)
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 2), "2")
	time.Sleep(5 * time.Millisecond)
	if len(pClient.subscriptions.edsMap) != 0 {
		t.Error("EdsMap length should be equal to 0. Got", len(pClient.subscriptions.edsMap))
	}
}

// TestMultipleSamewatchers simulates cases when several watchers are added on the same name.
func TestMultipleSamewatchers(t *testing.T) {
	adsMock := newADSServerMock(t, "")
	defer adsMock.Shutdown()
	pClient, err := newPilotClient(context.Background(), PilotClientOptions{
		PilotURL:    adsMock.Address(),
		IP:          "127.0.0.1",
		AppName:     "testAgent",
		Namespace:   "testns",
		DialOptions: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3 * time.Second)},
	})
	if err != nil {
		t.Fatal("Cannot start pilot client " + err.Error())
	}
	defer pClient.Shutdown()
	helloChannel, err := pClient.AddWatch("hello-service", "default", "6565")
	helloChannelMsgCounter := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case <-helloChannel:
				helloChannelMsgCounter++
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	helloChannel2, err := pClient.AddWatch("hello-service", "default", "6565")
	helloChannel2MsgCounter := 0
	go func(ctx context.Context) {
		for {
			select {
			case <-helloChannel2:
				helloChannel2MsgCounter++
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	if len(pClient.subscriptions.watchedList()) != 1 {
		t.Error("Number of different watchers should be equal to 1")
	}
	if len(pClient.subscriptions.watchers["outbound|6565||hello-service.default.svc.cluster.local"]) != 2 {
		t.Error("Number of stream for watch should be equal to 1")
	}
	time.Sleep(5 * time.Millisecond) // giving ads server some time
	if len(adsMock.streams) != 1 {
		t.Fatal("There must be 1 connection open to the ADS server. Got", len(adsMock.streams))
	}
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 1), "1")
	time.Sleep(5 * time.Millisecond)
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 2), "2")
	time.Sleep(5 * time.Millisecond)
	if len(pClient.subscriptions.edsMap) != 1 {
		t.Error("EdsMap length should be equal to 1. Got", len(pClient.subscriptions.edsMap))
	}
	if helloChannelMsgCounter != 2 {
		t.Error("Number of message on helloChannel should be 2. Got", helloChannelMsgCounter)
	}
	if helloChannel2MsgCounter != 2 {
		t.Error("Number of message on helloChannel2 should be 2. Got", helloChannel2MsgCounter)
	}
}

// estBackOff tests backoff function of a pilotClient
func TestBackoff(t *testing.T) {
	pClient := testMakePilotClient()
	if pClient.backoff(0) != pClient.options.InitialBackoff {
		t.Error("backoff with 0 retries should be options.InitialBackoff")
	}
	backoff0 := pClient.backoff(0)
	backoff1 := pClient.backoff(1)
	if backoff0 >= backoff1 {
		t.Errorf("With default options, backoff(0) %f should be lower than backoff(1) %f", float64(backoff0), float64(backoff1))
	}
}

// TestReconnection tests a pilot client can successfully reconnect to a
func TestReconnection(t *testing.T) {
	adsMock := newADSServerMock(t, "")
	pClient, err := newPilotClient(context.Background(), PilotClientOptions{
		PilotURL:       adsMock.Address(),
		IP:             "127.0.0.1",
		AppName:        "testAgent",
		Namespace:      "testns",
		InitialBackoff: 10 * time.Millisecond, // we don't want to enter backoff mechanism.
		MaxBackoff:     10 * time.Millisecond, // we don't want to enter backoff mechanism.
		DialOptions:    []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3 * time.Second)},
	})
	if err != nil {
		t.Fatal("Cannot start pilot client " + err.Error())
	}
	defer pClient.Shutdown()
	helloChannel, err := pClient.AddWatch("hello-service", "default", "6565")
	helloChannelMsgCounter := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case msg := <-helloChannel:
				helloChannelMsgCounter++
				// Test on message content
				if helloChannelMsgCounter == 1 {
					if len(msg) != 3 {
						t.Error("Expected message length was 3. Got ", len(msg))
					}
					tt := map[int][]string{
						0: []string{"10.4.0.16", "europe-west1", "europe-west1-b"},
						1: []string{"10.4.1.18", "europe-west1", "europe-west1-b"},
						2: []string{"10.4.2.2", "europe-west1", "europe-west1-d"},
					}
					for k, v := range tt {
						if msg[k].IP != v[0] || msg[k].Region != v[1] || msg[k].Zone != v[2] {
							t.Errorf("Test %d: Unexpected message. Expected %s got %s", k, v, msg)
						}
					}

				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	time.Sleep(5 * time.Millisecond) // giving ads server some time
	if len(adsMock.streams) != 1 {
		t.Fatal("There must be 1 connection open to the ADS server. Got", len(adsMock.streams))
	}
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 1), "1")
	time.Sleep(5 * time.Millisecond)
	if helloChannelMsgCounter != 1 {
		t.Error("Number of message on helloChannel should be 1. Got", helloChannelMsgCounter)
	}
	time.Sleep(50 * time.Millisecond)
	adsMock.Shutdown()
	time.Sleep(1 * time.Second)
	adsMock = newADSServerMock(t, ":"+strings.Split(pClient.options.PilotURL, ":")[1]) // use same port as previous server
	time.Sleep(2 * time.Second)
	if len(adsMock.streams) < 1 {
		t.Fatal("Pilot client has not reconnected in 2 seconds")
	}
	adsMock.streams[0].control <- endpointDiscoveryResponse(testCreateEdsMap(t, 1), "1")
	time.Sleep(5 * time.Millisecond)
	if helloChannelMsgCounter != 2 {
		t.Error("Number of message on helloChannel should be 2. Got", helloChannelMsgCounter)
	}
	defer adsMock.Shutdown()
}
