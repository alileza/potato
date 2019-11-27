package agent

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	dockerclient "github.com/docker/docker/client"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/alileza/potato/pb"
)

type Agent struct {
	log          *logrus.Logger
	dockerClient *dockerclient.Client

	ID               string
	ListenAddress    string
	AdvertiseAddress string
}

func NewAgent(
	logger *logrus.Logger,
	dockerClient *dockerclient.Client,
	ID string,
	listenAddress string,
	advertiseAddress string,
) *Agent {
	return &Agent{
		log:              logger,
		dockerClient:     dockerClient,
		ID:               ID,
		ListenAddress:    listenAddress,
		AdvertiseAddress: advertiseAddress,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	entry := a.log.WithField("potato", "agent")
	t := time.NewTicker(time.Second)
	defer t.Stop()

	conn, err := grpc.DialContext(
		ctx,
		a.ListenAddress,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(entry)),
	)
	if err != nil {
		return err
	}

	client := pb.NewPotatoClient(conn)

	var g run.Group

	a.log.Info("Initiating docker swarm request")
	resp, err := a.dockerClient.SwarmInit(ctx, swarm.InitRequest{
		ListenAddr:      a.ListenAddress,
		ForceNewCluster: true,
	})
	if err != nil {
		return err
	}
	a.log.Info(resp)

	g.Add(func() error {
		for range t.C {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			result, err := client.GetStatus(ctx, &pb.Status{Id: a.ID})
			if err != nil {
				a.log.Errorf("Failed to get status: %v", err)
				continue
			}
			expectedServicesMap := make(map[string]struct{})
			for _, svc := range result.GetServices() {
				expectedServicesMap[svc.GetImage()] = struct{}{}
			}

			runningServices, err := a.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
			if err != nil {
				a.log.Errorf("Failed to load servicesList: %v", err)
				continue
			}

			runningServicesMap := make(map[string]struct{})
			for _, svc := range runningServices {
				runningServicesMap[svc.Spec.TaskTemplate.ContainerSpec.Image] = struct{}{}
			}

			for _, service := range runningServices {
				image := service.Spec.TaskTemplate.ContainerSpec.Image
				// if image exist on expected, then we want to keep it
				if _, ok := expectedServicesMap[image]; ok {
					continue
				}

				a.log.Infof("Removing services for image: %s", image)
				if err := a.dockerClient.ServiceRemove(ctx, service.ID); err != nil {
					a.log.Errorf("service-remove: %v", err)
				}
			}

			// starting expected service if not yet started
			for _, service := range result.GetServices() {
				if _, ok := runningServicesMap[service.GetImage()]; ok {
					continue
				}

				a.log.Infof("Starting %s service", service.GetImage())
				if err := a.createService(ctx, service); err != nil {
					a.log.Errorf("Failed to create image: %v", err)
					continue
				}
			}

		}
		return nil
	}, a.logError)

	g.Add(func() error {
		return http.ListenAndServe(a.AdvertiseAddress, promhttp.Handler())
	}, a.logError)

	a.log.Infof("🥔 Agent connecting to %s\n  Advertising metrics to %s", a.ListenAddress, a.AdvertiseAddress)
	return g.Run()
}

func (a *Agent) createService(ctx context.Context, service *pb.Service) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var portConfig []swarm.PortConfig
	for _, port := range service.GetPorts() {
		p := strings.Split(port, ":")

		targetPort, err := strconv.ParseInt(p[0], 10, 32)
		if err != nil {
			return fmt.Errorf("Failed to parse target port: %v", err)
		}
		if len(p) == 1 {
			p = append(p, p[0])
		}

		publishedPort, err := strconv.ParseInt(p[1], 10, 32)
		if err != nil {
			return fmt.Errorf("Failed to parse published port: %v", err)
		}

		portConfig = append(portConfig, swarm.PortConfig{
			Protocol:      "tcp",
			TargetPort:    uint32(targetPort),
			PublishedPort: uint32(publishedPort),
		})
	}

	resp, err := a.dockerClient.ServiceCreate(ctx, swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image: service.GetImage(),
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Ports: portConfig,
		},
		Mode: swarm.ServiceMode{
			Replicated: &swarm.ReplicatedService{
				Replicas: &service.Replicas,
			},
		},
	}, types.ServiceCreateOptions{})
	if err != nil {
		return err
	}

	a.log.Infof("Successfully create service : %s", resp.ID)

	return nil
}

func (a *Agent) logError(err error) {
	a.log.Error(err)
}
