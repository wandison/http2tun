package main

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
)

const (
	_port = ":2345" // the incoming address for this agent, you can use docker -p to map ports
)

var (
	peer_conn *grpc.ClientConn
)

func main() {
	// resolve address & start listening
	tcpAddr, err := net.ResolveTCPAddr("tcp4", _port)
	checkError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	log.Println("listening on:", listener.Addr())

	c, err := grpc.Dial("127.0.0.1:1234", grpc.WithInsecure())
	checkError(err)
	peer_conn = c

	// loop accepting
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("accept failed:", err)
			continue
		}
		go handleClient(conn) // start a goroutine for every incoming connection for reading
	}
}

func peer(stream TunService_StreamClient, sess_die chan struct{}) <-chan []byte {
	ch := make(chan []byte)

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				log.Println(err)
				return
			}
			if err != nil {
				log.Println(err)
				return
			}
			select {
			case ch <- in.Message:
			case <-sess_die:
			}
		}
	}()
	return ch
}

func client(conn *net.TCPConn, sess_die chan struct{}) <-chan []byte {
	ch := make(chan []byte)
	go func() {
		for {
			bts := make([]byte, 512)
			n, err := conn.Read(bts)
			if err != nil {
				log.Println(err)
				return
			}

			select {
			case ch <- bts[:n]:
			case <-sess_die:
			}
		}
	}()
	return ch
}

func handleClient(conn *net.TCPConn) {
	cli := NewTunServiceClient(peer_conn)
	stream, err := cli.Stream(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	sess_die := make(chan struct{})
	ch_peer := peer(stream, sess_die)
	ch_client := client(conn, sess_die)
	defer func() {
		close(sess_die)
	}()

	for {
		select {
		case bts := <-ch_peer:
			log.Println(bts)
			if _, err := conn.Write(bts); err != nil {
				log.Println(err)
				return
			}
		case bts := <-ch_client:
			log.Println(bts)
			if err := stream.Send(&Tun_Frame{bts}); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
func checkError(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}