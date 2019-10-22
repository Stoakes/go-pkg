# Pilot discovery

_Subscribe to an Istio Pilot service and get available endpoints for a dns name._

## Usage

```go
/*
 * !! Pseudo-code
 */

import (
	"github.com/Stoakes/go-pkg/log"
	lb "github.com/Stoakes/go-pkg/grpc/pilotdiscovery"
	"github.com/Stoakes/go-pkg/grpc/pilotresolver"
)

func main() {
		pilotClient, err := lb.NewPilotClient(ctx, lb.PilotClientOptions{
			PilotURL: "localhost:15010", // assuming port-forward to istio-pilot on port 15010
			IP: "192.168.10.10",
			AppName: "front-service",
			Namespace: "default"})
		if err != nil {
			log.For(ctx).Fatalf("Cannot subscribe to pilot: %v", err)
		}
		pilotresolver.SetPilotClient(pilotClient)
		log.For(ctx).Info("Pilot subscription started")

		helloConnection, err := grpc.Dial(viper.GetString("hello-service"), []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.WithTimeout(10 * time.Second),
			grpc.WithBalancer(grpc.RoundRobin(&pilotresolver.Resolver{})),
        }...)
        if err != nil {
			log.For(ctx).Fatalf("Connection to hello-service error :%v", err)
		}
        helloClient := pb.NewGreeterClient(helloConnection)

		// Calls to helloClient will be load balanced accross every endpoints received via Pilot

		// You can also debug current state of Pilot client and known endpoints by exposing following router
		router := mux.NewRouter()
		router.HandleFunc("/debug/edsz", pilotClient.EdszHandler)
}
```
