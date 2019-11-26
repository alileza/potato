package agent

import (
	"context"
	"net/http"
	"time"

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
	log *logrus.Logger

	ID               string
	ListenAddress    string
	AdvertiseAddress string
}

func NewAgent(logger *logrus.Logger, ID, listenAddress, advertiseAddress string) *Agent {
	return &Agent{
		log:              logger,
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

	dockerClient, err := dockerclient.NewEnvClient()
	if err != nil {
		return err
	}

	a.log.Info("Initiating docker swarm request")
	resp, err := dockerClient.SwarmInit(ctx, swarm.InitRequest{
		ListenAddr:      a.ListenAddress,
		ForceNewCluster: true,
	})
	if err != nil {
		return err
	}
	a.log.Info(resp)

	client := pb.NewPotatoClient(conn)

	var g run.Group

	g.Add(func() error {
		for range t.C {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			plan, err := client.Status(ctx, &pb.Plan{Id: a.ID})
			if err != nil {
				a.log.Errorf("Failed to get status: %v", err)
				continue
			}

			a.log.Info(plan)
		}
		return nil
	}, a.logError)

	g.Add(func() error {
		return http.ListenAndServe(a.AdvertiseAddress, promhttp.Handler())
	}, a.logError)

	a.log.Infof("🥔 Agent connecting to %s\n  Advertising metrics to %s", a.ListenAddress, a.AdvertiseAddress)
	return g.Run()
}

func (a *Agent) logError(err error) {
	a.log.Error(err)
}
