package main

import (
	"google.golang.org/grpc"
	"net"
	"os"
)

const (
	_port = ":1234"
)

func main() {
	lis, err := net.Listen("tcp", _port)
	if err != nil {
		log.Critical(err)
		os.Exit(-1)
	}
	log.Info("listening on ", lis.Addr())

	s := grpc.NewServer()
	ins := new(server)
	pb.RegisterTunServiceServer(s, ins)
	s.Serve(lis)
}
