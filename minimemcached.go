package minimemcached

import (
	"bufio"
	gobytes "bytes"
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/benbjohnson/clock"
	"github.com/rs/zerolog/log"
)

const (
	Version = "1.0.1"
)

type MiniMemcached struct {
	server   *Server
	mu       sync.RWMutex
	items    map[string]*Item
	CASToken uint64
	port     uint16
	clock    clock.Clock
}

// Config contains minimum attributes to run mini-memcached.
// TODO(@sang-w0o): Selectively accept log writer and log level.
type Config struct {
	// Port is the port number where mini-memcached will start.
	// When given 0, mini-memcached will start running on a random available port.
	Port uint16
}

// Item is an object stored in mini-memcached.
type Item struct {
	// Value is the actual data stored in the item.
	Value []byte
	// Flags is a 32-bit unsigned integer that mini-memcached stores with the data
	// provided by the user.
	Flags uint32
	// Expiration is the expiration time in seconds.
	// 0 means no delay. IF Expiration is more than 30 days, mini-memcached
	// uses it as a UNIX timestamp for expiration.
	Expiration int32
	// CASToken is a unique unsigned 64-bit value of an existing item.
	CASToken uint64
	// createdAt is UNIX timestamp of the time when item has been created.
	// It is used for invalidations along with Expiration.
	createdAt int64
}

type Option func(m *MiniMemcached)

// newMiniMemcached returns a newMiniMemcached, non-started, MiniMemcached object.
func newMiniMemcached(opts ...Option) *MiniMemcached {
	m := MiniMemcached{
		items:    map[string]*Item{},
		CASToken: 0,
		clock:    clock.New(),
	}

	for _, opt := range opts {
		opt(&m)
	}

	return &m
}

// WithClock applies custom Clock interface. Clock will be used when Item is created
func WithClock(clk clock.Clock) Option {
	return func(m *MiniMemcached) {
		m.clock = clk
	}
}

// Run creates and starts a MiniMemcached server on a random, available port.
// Close with Close().
func Run(cfg *Config, opts ...Option) (*MiniMemcached, error) {
	m := newMiniMemcached(opts...)
	return m, m.start(cfg.Port)
}

// Close closes mini-memcached server and clears all objects stored.
func (m *MiniMemcached) Close() {
	log.Info().Msg("closed mini-memcached.")
	m.mu.Lock()
	m.items = nil
	m.server.close()
	m.mu.Unlock()
}

func (m *MiniMemcached) Port() uint16 {
	return m.port
}

// Start starts a mini-memcached server. It listens on a given port.
func (m *MiniMemcached) start(port uint16) error {
	s, err := newServer(port)
	if err != nil {
		return err
	}

	tcpAddr, ok := s.l.Addr().(*net.TCPAddr)
	if !ok {
		return errors.New("failed to obtain tcp address")
	}

	m.port = uint16(tcpAddr.Port)
	m.server = s
	m.newServer()
	return nil
}

func (m *MiniMemcached) newServer() {
	go func() {
		m.serve()
	}()
}

func (m *MiniMemcached) serve() {
	for {
		conn, err := m.server.l.Accept()
		if err != nil {
			return
		}
		go m.serveConn(conn)
	}
}

func (m *MiniMemcached) serveConn(conn net.Conn) {
	for {
		reader := bufio.NewReader(conn)
		req, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			continue
		}

		if err != nil {
			log.Err(err).Msgf("err reading string: %v", err)
			return
		}

		req = strings.TrimSuffix(req, "\r\n")
		cmdLine := strings.Split(req, " ")
		cmd := strings.ToLower(cmdLine[0])
		switch cmd {
		case GET:
			handleGet(m, cmdLine, conn)
		case GETS:
			handleGets(m, cmdLine, conn)
		case SET:
			value, err := reader.ReadBytes('\n')
			if err != nil {
				handleErr(conn)
			}

			value = gobytes.TrimSuffix(value, crlf)
			handleSet(m, cmdLine, value, conn)
		case ADD:
			value, err := reader.ReadBytes('\n')
			if err != nil {
				handleErr(conn)
			}

			value = gobytes.TrimSuffix(value, crlf)
			handleAdd(m, cmdLine, value, conn)
		case REPLACE:
			value, err := reader.ReadBytes('\n')
			if err != nil {
				handleErr(conn)
			}

			value = gobytes.TrimSuffix(value, crlf)
			handleReplace(m, cmdLine, value, conn)
		case APPEND:
			value, err := reader.ReadBytes('\n')
			if err != nil {
				handleErr(conn)
			}

			value = gobytes.TrimSuffix(value, crlf)
			handleAppend(m, cmdLine, value, conn)
		case PREPEND:
			value, err := reader.ReadBytes('\n')
			if err != nil {
				handleErr(conn)
			}

			value = gobytes.TrimSuffix(value, crlf)
			handlePrepend(m, cmdLine, value, conn)
		case DELETE:
			handleDelete(m, cmdLine, conn)
		case INCR:
			handleIncr(m, cmdLine, conn)
		case DECR:
			handleDecr(m, cmdLine, conn)
		case TOUCH:
			handleTouch(m, cmdLine, conn)
		case FLUSH_ALL:
			handleFlushAll(m, conn)
		case CAS:
			value, err := reader.ReadBytes('\n')
			if err != nil {
				handleErr(conn)
			}

			value = gobytes.TrimSuffix(value, crlf)
			handleCas(m, cmdLine, value, conn)
		case VERSION:
			handleVersion(m, conn)
		default:
			handleErr(conn)
		}
	}
}

// invalidate() invalidates objects by its expiration value.
func (m *MiniMemcached) invalidate(key string) {
	currentTimestamp := m.clock.Now().Unix()
	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.items[key]
	if item == nil {
		return
	}
	if item.Expiration == 0 {
		return
	}
	if item.Expiration > ttlUnixTimestamp {
		if currentTimestamp > int64(item.Expiration) {
			delete(m.items, key)
			return
		}
		return
	}
	if currentTimestamp-item.createdAt >= int64(item.Expiration) {
		delete(m.items, key)
		return
	}
}
