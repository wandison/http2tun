package main

import (
	"crypto/rc4"
	"io"
	"log"
	"net"
)

const (
	_key_recv = "zkxkiej!@#$"
	_key_send = "*!@#($JZVAS"
)

type server struct{}

// stream receiver
func (s *server) recv(stream TunService_StreamServer, sess_die chan struct{}) chan []byte {
	ch := make(chan []byte)
	go func() {
		defer func() {
			close(ch)
		}()

		decoder, err := rc4.NewCipher([]byte(_key_recv))
		if err != nil {
			log.Println(err)
			return
		}

		for {
			in, err := stream.Recv()
			if err == io.EOF { // client closed
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

// endpoint receiver
func (s *server) endpoint(sess_die chan struct{}) (c net.Conn, ch_endpoint <-chan []byte) {
	ch := make(chan []byte)
	conn, err := net.Dial("tcp", "localhost:1194")
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		defer func() {
			close(ch)
		}()
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
	return conn, ch
}

// stream server
func (s *server) Stream(stream TunService_StreamServer) error {
	sess_die := make(chan struct{})
	ch_agent := s.recv(stream, sess_die)
	conn, ch_endpoint := s.endpoint(sess_die)
	if conn == nil {
		return nil
	}

	defer func() {
		close(sess_die)
	}()

	encoder, err := rc4.NewCipher([]byte(_key_send))
	if err != nil {
		log.Println(err)
		return nil
	}

	for {
		select {
		case bts, ok := <-ch_agent:
			if !ok {
				return nil
			}
			conn.Write(bts)
		case bts, ok := <-ch_endpoint:
			if !ok {
				return nil
			}
			encoder.XORKeyStream(bts, bts)
			if err := stream.Send(&Tun_Frame{bts}); err != nil {
				log.Println(err)
				return err
			}
		}
	}
}
