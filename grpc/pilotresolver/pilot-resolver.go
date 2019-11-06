package pilotresolver

import (
	"fmt"
	"strings"
	"time"

	"github.com/Stoakes/go-pkg/log"
	"github.com/Stoakes/go-pkg/pilotdiscovery"

	"google.golang.org/grpc/naming"
)

var pilotClient pilotdiscovery.PilotClient

// SetPilotClient sets the pilotresolver pkg variable
func SetPilotClient(pClient pilotdiscovery.PilotClient) {
	pilotClient = pClient
}

// Resolver implements the naming.Resolver interface.
// WARNING: There must be one Resolver instantiated per connected service.
// That is to say: do not use shared grpc.DialOptions.
type Resolver struct{}

type pilotWatcher struct {
	target        string // the hostname:port to connect to
	port          string
	addresses     map[string]bool                    // Stores the known endpoints for <target>
	endpointsChan chan pilotdiscovery.EndpointsState // incoming channel of endpoints from pilot client
	updateChan    chan []*naming.Update              // outgoing channel of naming update for the gRPC client
	errChan       chan error                         // channel for errors. Every error posted to that channel will be logged
	quit          chan bool                          // channel for stop signal. Any boolean posted on it will stop the watcher
}

// Resolve creates a Watcher for target.
// target is expected to match <service-name>.<namespace>(.<whatever>):<port>
// Because of Resolve interface, pilotClient must be a package variable
func (c *Resolver) Resolve(target string) (naming.Watcher, error) {
	if pilotClient == nil {
		return nil, fmt.Errorf("PilotClient not initialized. Use SetPilotClient prior to any operation")
	}
	log.Bg().Info("Starting look aside resolver for " + target)
	return newPilotWatcher(target), nil
}

func newPilotWatcher(target string) *pilotWatcher {
	watcher := &pilotWatcher{
		addresses:  map[string]bool{},
		target:     target,
		updateChan: make(chan []*naming.Update, 1),
		errChan:    make(chan error),
		quit:       make(chan bool),
	}

	// Remove the port from the string
	str := strings.Split(target, ":")
	if len(str) != 2 {
		panic(fmt.Sprintf("gRPC connection target %s cannot be split into host:port", target))
	}
	svcDNSName := str[0]
	watcher.port = str[1]

	str = strings.Split(svcDNSName, ".")
	if len(str) < 2 {
		panic(fmt.Sprintf("gRPC service name %s cannot be split into service.namespace", svcDNSName))
	}

	var err error
	watcher.endpointsChan, err = pilotClient.AddWatch(str[0], str[1], watcher.port)
	if err != nil {
		panic(fmt.Sprintf("Cannot add a watch on %s.%s:%s : %s", str[0], str[1], watcher.port, err))
	}

	go watcher.watch()
	return watcher
}

// Next blocks until an update or error happens. It may return one or more
// updates. The first call should get the full set of the results. It should
// return an error if and only if Watcher cannot recover.
func (c *pilotWatcher) Next() ([]*naming.Update, error) {
	for {
		select {
		case err := <-c.errChan:
			// GRPC doesn't handle the thrown error here.  It just sits idly and
			// throws a log message saying it is stopping.  We don't want that behavior.
			// If there is an error returned from the DNS service, we simply continue and
			// log the error.  The drawback is that any new pods that spin up will not
			// be picked up by the service
			log.Bg().Errorf("Got DNS Error: %v", err)
		case update := <-c.updateChan:
			return update, nil
		default:
			time.Sleep(time.Second)
		}
	}
}

// Close closes the Watcher.
func (c *pilotWatcher) Close() {
	c.quit <- true
}

func (c *pilotWatcher) watch() {
	for {
		select {
		case <-c.quit:
			return
		case endpoints := <-c.endpointsChan:
			log.Bg().Debugf("Pilot listener reponse for %s: %v", c.target, endpoints)
			updates := []*naming.Update{}
			for k := range c.addresses {
				c.addresses[k] = false
			}

			for _, v := range endpoints.Endpoints {
				if _, ok := c.addresses[v.IP]; !ok {
					updates = append(updates, &naming.Update{
						Op:   naming.Add,
						Addr: v.IP + ":" + c.port,
					})
				}
				c.addresses[v.IP] = true
			}
			for k, v := range c.addresses {
				if !v {
					updates = append(updates, &naming.Update{
						Op:   naming.Delete,
						Addr: k + ":" + c.port,
					})
					delete(c.addresses, k)
				}
			}
			log.Bg().Debugf("Posting %d endpoints updates for target %s", len(updates), c.target)
			c.updateChan <- updates
		}
	}
}
