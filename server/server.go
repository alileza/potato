package server

import (
	"context"
	"net"
	"net/http"
	"strings"

	pb "github.com/alileza/potato/pb"
	"github.com/jmoiron/sqlx"
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
	databaseDSN   string
}

func NewServer(logger *logrus.Logger, listenAddress, databaseDSN string) *Server {
	entry := logger.WithField("potato", "server")

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc_middleware.WithUnaryServerChain(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_logrus.UnaryServerInterceptor(entry),
		),
	}

	srv := grpc.NewServer(opts...)

	return &Server{
		srv:           srv,
		log:           logger,
		listenAddress: listenAddress,
		databaseDSN:   databaseDSN,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	conn, err := sqlx.Open("postgres", s.databaseDSN)
	if err != nil {
		return err
	}
	defer conn.Close()

	pb.RegisterPotatoServer(s.srv, &PotatoServer{conn})

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

type PotatoServer struct {
	DB *sqlx.DB
}

func (p *PotatoServer) GetStatus(ctx context.Context, in *pb.Status) (*pb.Status, error) {
	tx, err := p.DB.PreparexContext(ctx, `SELECT version, ports, replicas FROM releases WHERE hostname=$1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	var result []struct {
		Version  string `db:"version"`
		Ports    string `db:"ports"`
		Replicas int64  `db:"replicas"`
	}

	if err := tx.SelectContext(ctx, &result, in.GetId()); err != nil {
		return nil, err
	}

	response := &pb.Status{
		Id:       in.GetId(),
		Services: []*pb.Service{},
	}

	for _, res := range result {
		response.Services = append(response.Services, &pb.Service{
			Image:    res.Version,
			Replicas: uint64(res.Replicas),
			Ports:    strings.Split(res.Ports, ";"),
		})
	}

	return response, nil
}

func (s *Server) logError(err error) {
	s.log.Error(err)
}
