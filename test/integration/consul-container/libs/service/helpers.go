package service

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	libcluster "github.com/hashicorp/consul/test/integration/consul-container/libs/cluster"
	"github.com/hashicorp/consul/test/integration/consul-container/libs/utils"
)

const (
	StaticServerServiceName  = "static-server"
	StaticServer2ServiceName = "static-server-2"
	StaticClientServiceName  = "static-client"
)

type Checks struct {
	Name string
	TTL  string
}

type SidecarService struct {
	Port int
}

type ServiceOpts struct {
	Name     string
	ID       string
	Meta     map[string]string
	HTTPPort int
	GRPCPort int
	Checks   Checks
	Connect  SidecarService
}

// createAndRegisterStaticServerAndSidecar register the services and launch static-server containers
func createAndRegisterStaticServerAndSidecar(node libcluster.Agent, grpcPort int, svc *api.AgentServiceRegistration, containerArgs ...string) (Service, Service, error) {
	// Do some trickery to ensure that partial completion is correctly torn
	// down, but successful execution is not.
	var deferClean utils.ResettableDefer
	defer deferClean.Execute()

	if err := node.GetClient().Agent().ServiceRegister(svc); err != nil {
		return nil, nil, err
	}

	// Create a service and proxy instance
	serverService, err := NewExampleService(context.Background(), svc.ID, svc.Port, grpcPort, node, containerArgs...)
	if err != nil {
		return nil, nil, err
	}
	deferClean.Add(func() {
		_ = serverService.Terminate()
	})

	serverConnectProxy, err := NewConnectService(context.Background(), fmt.Sprintf("%s-sidecar", svc.ID), svc.ID, []int{svc.Port}, node) // bindPort not used
	if err != nil {
		return nil, nil, err
	}
	deferClean.Add(func() {
		_ = serverConnectProxy.Terminate()
	})

	// disable cleanup functions now that we have an object with a Terminate() function
	deferClean.Reset()

	return serverService, serverConnectProxy, nil
}

func CreateAndRegisterStaticServerAndSidecar(node libcluster.Agent, serviceOpts *ServiceOpts, containerArgs ...string) (Service, Service, error) {
	// Register the static-server service and sidecar first to prevent race with sidecar
	// trying to get xDS before it's ready
	req := &api.AgentServiceRegistration{
		Name: serviceOpts.Name,
		ID:   serviceOpts.ID,
		Port: serviceOpts.HTTPPort,
		Connect: &api.AgentServiceConnect{
			SidecarService: &api.AgentServiceRegistration{
				Proxy: &api.AgentServiceConnectProxyConfig{},
			},
		},
		Check: &api.AgentServiceCheck{
			Name:     "Static Server Listening",
			TCP:      fmt.Sprintf("127.0.0.1:%d", serviceOpts.HTTPPort),
			Interval: "10s",
			Status:   api.HealthPassing,
		},
		Meta: serviceOpts.Meta,
	}
	return createAndRegisterStaticServerAndSidecar(node, serviceOpts.GRPCPort, req, containerArgs...)
}

func CreateAndRegisterStaticServerAndSidecarWithChecks(node libcluster.Agent, serviceOpts *ServiceOpts) (Service, Service, error) {
	// Register the static-server service and sidecar first to prevent race with sidecar
	// trying to get xDS before it's ready
	req := &api.AgentServiceRegistration{
		Name: serviceOpts.Name,
		ID:   serviceOpts.ID,
		Port: serviceOpts.HTTPPort,
		Connect: &api.AgentServiceConnect{
			SidecarService: &api.AgentServiceRegistration{
				Proxy: &api.AgentServiceConnectProxyConfig{},
				Port:  serviceOpts.Connect.Port,
			},
		},
		Checks: api.AgentServiceChecks{
			{
				Name: serviceOpts.Checks.Name,
				TTL:  serviceOpts.Checks.TTL,
			},
		},
		Meta: serviceOpts.Meta,
	}

	return createAndRegisterStaticServerAndSidecar(node, serviceOpts.GRPCPort, req)
}

func CreateAndRegisterStaticClientSidecar(
	node libcluster.Agent,
	peerName string,
	localMeshGateway bool,
) (*ConnectContainer, error) {
	// Do some trickery to ensure that partial completion is correctly torn
	// down, but successful execution is not.
	var deferClean utils.ResettableDefer
	defer deferClean.Execute()

	mgwMode := api.MeshGatewayModeRemote
	if localMeshGateway {
		mgwMode = api.MeshGatewayModeLocal
	}

	// Register the static-client service and sidecar first to prevent race with sidecar
	// trying to get xDS before it's ready
	req := &api.AgentServiceRegistration{
		Name: StaticClientServiceName,
		Port: 8080,
		Connect: &api.AgentServiceConnect{
			SidecarService: &api.AgentServiceRegistration{
				Proxy: &api.AgentServiceConnectProxyConfig{
					Upstreams: []api.Upstream{{
						DestinationName:  StaticServerServiceName,
						DestinationPeer:  peerName,
						LocalBindAddress: "0.0.0.0",
						LocalBindPort:    libcluster.ServiceUpstreamLocalBindPort,
						MeshGateway: api.MeshGatewayConfig{
							Mode: mgwMode,
						},
					}},
				},
			},
		},
	}

	if err := node.GetClient().Agent().ServiceRegister(req); err != nil {
		return nil, err
	}

	// Create a service and proxy instance
	clientConnectProxy, err := NewConnectService(context.Background(), fmt.Sprintf("%s-sidecar", StaticClientServiceName), StaticClientServiceName, []int{libcluster.ServiceUpstreamLocalBindPort}, node)
	if err != nil {
		return nil, err
	}
	deferClean.Add(func() {
		_ = clientConnectProxy.Terminate()
	})

	// disable cleanup functions now that we have an object with a Terminate() function
	deferClean.Reset()

	return clientConnectProxy, nil
}
