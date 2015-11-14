package main

import (
	"errors"
	"io"
	"strconv"
	"time"
)

type server struct{}

// stream receiver
func (s *server) recv(stream GameService_StreamServer, sess_die chan struct{}) chan *Tun_Frame {
	ch := make(chan *Game_Frame, 1)
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

// stream server
func (s *server) Stream(stream GameService_StreamServer) error {
	// session init
	var sess Session
	sess_die := make(chan struct{})
	ch_agent := s.recv(stream, sess_die)

	defer func() {
		close(sess_die)
	}()

	// >> main message loop <<
	for {
		select {
		case frame, ok := <-ch_agent: // frames from agent
			if !ok { // EOF
				return nil
			}
		}
	}
}
