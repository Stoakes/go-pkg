package pilotdiscovery

import (
	"sync"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"go.uber.org/atomic"
)

// stateStore stores the state of the pilot Client: known internal subscriptions, known endpoints
type stateStore struct {
	lock     sync.Mutex                               // Lock on state modification
	watchers map[string][](chan EndpointsState)       // for a given cluster name, every channel to watchers on it
	edsMap   map[string]*xdsapi.ClusterLoadAssignment // for a given cluster name every currently known endpoints of it
	version  *atomic.Uint64                           // Mostly unused for now, number of modification since listener stared
}

// EndpointsState is the list of every known endpoint at a moment. This type aims at making output less misleading
// It is the "state of the world" as the AggregatedResources stream send them
// by opposition to sending the diff
type EndpointsState struct {
	Endpoints []Endpoint
}

// Endpoint is the representation of an endpoint and what will be sent to internal consumers
type Endpoint struct {
	IP     string // IP address of the endpoint
	Region string // Region of the IP (europe-west2)
	Zone   string //  zone/AZ of the IP (europe-west2-b)
}

// newStateStore creates a new state store
func newStateStore() *stateStore {
	return &stateStore{
		watchers: make(map[string][](chan EndpointsState)),
		version:  atomic.NewUint64(0),
		edsMap:   make(map[string]*xdsapi.ClusterLoadAssignment),
	}
}

// Lock locks the state store. To be used prior to any modification
func (s *stateStore) Lock() {
	s.lock.Lock()
}

// Unlock unlocks the state store
func (s *stateStore) Unlock() {
	s.lock.Unlock()
}

// addWatcher add a watcher on a clustername returns a channel where every ip
func (s *stateStore) addWatcher(host string, namespace string, port string) (chan EndpointsState, error) {
	s.Lock()
	defer s.Unlock()
	address := s.format(host, namespace, port)
	_, ok := s.watchers[address]
	if ok { // key alread exists in the watcher map: append to the list
		newChannel := make(chan EndpointsState)
		s.watchers[address] = append(s.watchers[address], newChannel)
		return newChannel, nil
	}
	// the address is not already watched, create the entry in the map
	newChannel := make(chan EndpointsState)
	s.watchers[address] = [](chan EndpointsState){newChannel}
	return newChannel, nil
}

// watchedList returns a list of watched clustername
// used when requesting querying Pilot
func (s *stateStore) watchedList() []string {
	s.Lock()
	defer s.Unlock()
	resp := make([]string, len(s.watchers))
	i := 0
	for name := range s.watchers {
		resp[i] = name
		i++
	}
	return resp
}

// format formats a tuple (host, namespace, port) into a pilot valid format
func (s *stateStore) format(host string, namespace string, port string) string {
	return "outbound|" + port + "||" + host + "." + namespace + ".svc.cluster.local"
}

// clear closes every channel and remove all state from stateStore
func (s *stateStore) clear() {
	s.Lock()
	defer s.Unlock()
	for _, watch := range s.watchers {
		for _, channel := range watch {
			close(channel)
		}
	}
	s.watchers = make(map[string][](chan EndpointsState))
	s.version = atomic.NewUint64(0)
	s.edsMap = make(map[string]*xdsapi.ClusterLoadAssignment)
}
