package agent

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/alileza/potato/pb"
)

type Agent struct {
	log *logrus.Logger

	ID        string
	ServerURL string
}

func NewAgent(logger *logrus.Logger, ID, serverURL string) *Agent {
	return &Agent{
		log:       logger,
		ID:        ID,
		ServerURL: serverURL,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	conn, err := grpc.DialContext(ctx, a.ServerURL, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := pb.NewPotatoClient(conn)

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
}
