package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	dispatch_echov1 "github.com/jon-whit/dispatch-echo/internal/proto/dispatch-echo/v1"
	hashring "github.com/jon-whit/grpc-hashring"
	"github.com/sercand/kuberesolver/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	flag.Parse()

	podIP, ok := os.LookupEnv("POD_IP")
	if !ok {
		log.Fatalf("POD_IP env must be set")
	}

	log.Printf("pod ip '%s'", podIP)

	kuberesolver.RegisterInCluster()

	var echoClient dispatch_echov1.DispatchEchoServiceClient

	s := &echoServer{
		serverID:       podIP,
		dispatchClient: echoClient,
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to start tcp listener on address '%s'", ":50051")
	}

	grpcServer := grpc.NewServer()
	dispatch_echov1.RegisterDispatchEchoServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				log.Fatalf("failed to start gRPC server: %v", err)
			}
		}
	}()

	addr := "kubernetes:///dispatch-echo.default:50051"
	conn, err := grpc.Dial(
		addr,
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [
		  {
		    "hashring": {
			  "keyReplicationCount": 2
		    }
		  }
		]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to establish grpc client connection to '%s'", addr)
	}
	defer conn.Close()

	s.SetEchoClient(dispatch_echov1.NewDispatchEchoServiceClient(conn))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
}

type echoServer struct {
	dispatch_echov1.UnimplementedDispatchEchoServiceServer
	dispatchClient dispatch_echov1.DispatchEchoServiceClient
	serverID       string
}

func (e *echoServer) SetEchoClient(c dispatch_echov1.DispatchEchoServiceClient) {
	e.dispatchClient = c
}

func (e *echoServer) Echo(
	ctx context.Context,
	req *dispatch_echov1.EchoRequest,
) (*dispatch_echov1.EchoResponse, error) {
	msg := req.GetMessage()

	if req.GetPeerId() == e.serverID {
		log.Printf("serving Echo \n")
		return &dispatch_echov1.EchoResponse{
			Message: msg,
			PeerId:  e.serverID,
		}, nil
	}

	ctx = hashring.ContextWithHashringKey(ctx, msg)

	log.Printf("dispatching Echo to peer\n")
	return e.dispatchClient.Echo(ctx, &dispatch_echov1.EchoRequest{
		Message: msg,
		PeerId:  e.serverID,
	})
}
