package minimemcached

import (
	"fmt"
	"log"
	"net"
)

type server struct {
	l net.Listener
}

// NewServer starts and returns a server listening on a given port.
func newServer(port uint16) (*server, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("failed to listen on port: %d", port)
		return nil, err
	}
	return &server{l: l}, nil
}

// Close closes a server started with NewServer().
func (s *server) close() {
	if s.l != nil {
		_ = s.l.Close()
		s.l = nil
	}
}
