package main

import (
	"crypto/rc4"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
)

const (
	_port     = ":1194" // the incoming address for this agent, you can use docker -p to map ports
	_key_send = "zkxkiej!@#$"
	_key_recv = "*!@#($JZVAS"
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

	c, err := grpc.Dial("vps:1234", grpc.WithInsecure())
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
		decoder, err := rc4.NewCipher([]byte(_key_recv))
		if err != nil {
			log.Println(err)
			return
		}

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
			decoder.XORKeyStream(in.Message, in.Message)
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
	defer func() {
		conn.Close()
	}()

	cli := NewTunServiceClient(peer_conn)
	stream, err := cli.Stream(context.Background())
	if err != nil {
		log.Println(err)
		return
	}
	encoder, err := rc4.NewCipher([]byte(_key_send))
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
			if _, err := conn.Write(bts); err != nil {
				log.Println(err)
				return
			}
		case bts := <-ch_client:
			encoder.XORKeyStream(bts, bts)
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
