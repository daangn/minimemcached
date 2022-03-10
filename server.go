package minimemcached

import (
	"fmt"
	"log"
	"net"
)

type Server struct {
	l net.Listener
}

// NewServer starts and returns a server listening on a given port.
func newServer(port uint16) (*Server, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("failed to listen on port: %d", port)
		return nil, err
	}
	return &Server{l: l}, nil
}

// Close closes a server started with NewServer().
func (s *Server) close() {
	if s.l != nil {
		_ = s.l.Close()
		s.l = nil
	}
}
