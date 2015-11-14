package main

import (
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	_port = ":1234"
)

func main() {
	lis, err := net.Listen("tcp", _port)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on ", lis.Addr())

	s := grpc.NewServer()
	ins := new(server)
	RegisterTunServiceServer(s, ins)
	s.Serve(lis)
}
