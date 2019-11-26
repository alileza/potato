package server

import (
	"context"
	"net"
	"net/http"

	pb "github.com/alileza/potato/pb"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
)

type Server struct {
	srv *grpc.Server
	log *logrus.Logger

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
		log:           logger,
		listenAddress: listenAddress,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	l, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return err
	}
	var (
		g run.Group
		m = cmux.New(l)
	)

	httpL := m.Match(cmux.HTTP1(), cmux.HTTP1Fast())
	grpcL := m.Match(cmux.HTTP2())

	g.Add(func() error {
		srv := &http.Server{Handler: promhttp.Handler()}
		return srv.Serve(httpL)
	}, s.logError)

	g.Add(func() error {
		return s.srv.Serve(grpcL)
	}, s.logError)

	g.Add(func() error {
		return m.Serve()
	}, s.logError)

	s.log.Infof("🥔 GRPC Server serving on %s", s.listenAddress)
	return g.Run()
}

type PotatoServer struct{}

func (p *PotatoServer) Status(ctx context.Context, plan *pb.Plan) (*pb.Plan, error) {
	return &pb.Plan{}, nil
}

func (s *Server) logError(err error) {
	s.log.Error(err)
}
