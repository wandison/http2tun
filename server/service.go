package main

import (
	"errors"
	"io"
	"strconv"
	"time"
)

type server struct{}

// stream receiver
func (s *server) recv(stream TunService_StreamServer, sess_die chan struct{}) chan *Tun_Frame {
	ch := make(chan *Tun_Frame, 1)
	go func() {
		defer func() {
			close(ch)
		}()
		for {
			in, err := stream.Recv()
			if err == io.EOF { // client closed
				log.Trace(err)
				return
			}

			if err != nil {
				log.Error(err)
				return
			}
			select {
			case ch <- in:
			case <-sess_die:
			}
		}
	}()
	return ch
}

// endpoint receiver
func (s *server) endpoint(sess_die chan struct{}) <-chan []byte {
	ch := make(chan []byte)
	go func() {
		conn, err := net.Dial("tcp", "localhost:8888")
		if err != nil {
			log.Error(err)
			return
		}
		bts := make([]byte, 512)
		n, err := conn.Read(bts)
		if err != nil {
			log.Error(err)
			return
		}

		select {
		case ch <- bts[:n]:
		case <-sess_die:
		}
	}()
}

// stream server
func (s *server) Stream(stream TunService_StreamServer) error {
	ch_agent := s.recv(stream, sess_die)
	ch_endpoint := s.endpoint(sess_die)
	defer func() {
		close(sess_die)
	}()
	for {
		select {
		case frame, ok := <-ch_agent: // frames from agent
			if !ok { // EOF
				return nil
			}
		case bts := <-ch_endpoint:
			if err := stream.Send(&Tun_Frame{bts}); err != nil {
				log.Critical(err)
				return err
			}
		}
	}
}
