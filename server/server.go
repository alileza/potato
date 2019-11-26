package server

import (
	"context"
	"log"
	"net"

	pb "github.com/alileza/potato/pb"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Server struct {
	srv *grpc.Server

	listenAddress string
}

func NewServer(logger *logrus.Logger, listenAddress string) *Server {
	entry := logger.WithField("potato", "server")

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc_middleware.WithUnaryServerChain(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_logrus.UnaryServerInterceptor(entry),
		),
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterPotatoServer(srv, &PotatoServer{})

	return &Server{
		srv:           srv,
		listenAddress: listenAddress,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	l, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return err
	}

	log.Printf("[INFO] GRPC Server serving on %s", s.listenAddress)
	return s.srv.Serve(l)
}

type PotatoServer struct{}

func (p *PotatoServer) Status(ctx context.Context, plan *pb.Plan) (*pb.Plan, error) {
	return &pb.Plan{}, nil
}
