package passage

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
)

func TestListen(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}

	// Create a Consul API client
	serverOnline := make(chan bool)

	wg.Add(1)
	go func() {
		client, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			panic(errors.Wrap(err, "could not init consul client"))
		}

		// Register service
		err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
			Name: "http-server",
			Port: 4389,
			Connect: &api.AgentServiceConnect{
				Native: true,
			},
		})
		defer func() {
			client.Agent().ServiceDeregister("http-server")
		}()
		if err != nil {
			panic(errors.Wrap(err, "could not register service"))
		}

		logrus.Debug("creating service")
		// Create an instance representing this service. "my-service" is the
		// name of _this_ service. The service should be cleaned up via Close.
		svc, _ := connect.NewService("http-server", client)
		defer svc.Close()
		<-svc.ReadyWait()

		// Creating an HTTP server that serves via Connect
		server := &http.Server{
			Addr:      ":4389",
			TLSConfig: svc.ServerTLSConfig(),
			Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(http.StatusOK)
				writer.Write([]byte("hello world"))
			}),
		}

		// Serve!
		logrus.Debug("starting server")
		go func() {
			err := server.ListenAndServeTLS("", "")
			if err != nil {
				panic(errors.Wrap(err, "could not start server"))
			}
		}()

		logrus.Debug("started server")
		serverOnline <- true
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		<-serverOnline
		//defer cancel()

		client, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			panic(errors.Wrap(err, "could not init consul client"))
		}

		// register client
		err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{Name: "http-client"})

		// connect as client
		svc, _ := connect.NewService("http-client", client)
		defer svc.Close()

		httpClient := svc.HTTPClient()

		logrus.Debug("starting request")
		response, err := httpClient.Get("https://http-server.service.consul")
		if err != nil {
			logrus.Error(errors.Wrap(err, "could not connect"))
			return
		}

		body, err := io.ReadAll(response.Body)
		logrus.Debug(string(body))
		wg.Done()
	}()

	wg.Wait()
}

func Test_TunnelClient(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(errors.Wrap(err, "could not init consul client"))
	}

	// register client
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{Name: "tunnel-client"})

	// connect as client
	svc, _ := connect.NewService("tunnel-client", client)
	defer svc.Close()

	logrus.Debug("starting request")
	conn, err := svc.Dial(context.Background(), &connect.ConsulResolver{
		Client: client,
		Name:   "tunnel_bf83a7ae-9507-4e09-81b8-a51abffc18f9",
	})
	if err != nil {
		panic(errors.Wrap(err, "could not dial"))
	}

	logrus.Debug("Making request")
	if _, err := conn.Write([]byte("GET /")); err != nil {
		panic(errors.Wrap(err, "could not get"))
	}

	bytes, err := io.ReadAll(conn)
	if err != nil {
		panic(errors.Wrap(err, "could not read bytes"))
	}
	logrus.Debug(string(bytes))
}
