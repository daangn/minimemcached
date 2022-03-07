package minimemcached

import "fmt"

const (
	GET       = "get"
	GETS      = "gets"
	CAS       = "cas"
	SET       = "set"
	TOUCH     = "touch"
	ADD       = "add"
	REPLACE   = "replace"
	APPEND    = "append"
	PREPEND   = "prepend"
	DELETE    = "delete"
	INCR      = "incr"
	DECR      = "decr"
	FLUSH_ALL = "flush_all"
	VERSION   = "version"
)

var (
	crlf                                   = []byte("\r\n")
	resultOK                               = []byte("OK\r\n")
	resultStored                           = []byte("STORED\r\n")
	resultTouched                          = []byte("TOUCHED\r\n")
	resultNotStored                        = []byte("NOT_STORED\r\n")
	resultNotFound                         = []byte("NOT_FOUND\r\n")
	resultDeleted                          = []byte("DELETED\r\n")
	resultExists                           = []byte("EXISTS\r\n")
	resultClientErrBadCliFormat            = []byte("CLIENT_ERROR bad command line format\r\n")
	resultClientErrBadDataChunk            = []byte("CLIENT_ERROR bad data chunk\r\n")
	resultClientErrIncrDecrNonNumericValue = []byte("CLIENT_ERROR cannot increment or decrement non-numeric value\r\n")
	resultClientErrInvalidNumericDeltaArg  = []byte("CLIENT_ERROR invalid numeric delta argument\r\n")
	resultClientErrInvalidExpTimeArg       = []byte("CLIENT_ERROR invalid exptime argument\r\n")
	resultEnd                              = []byte("END\r\n")
	resultErr                              = []byte("ERROR\r\n")
	resultNotImplementedCmdErr             = []byte("SERVER_ERROR command yet not implemented in mini-memcached.\r\n")
	resultVersion                          = []byte(fmt.Sprintf("VERSION mini-memcached %s\r\n", Version))
	value                                  = "VALUE"
)

const (
	maxKeyLength     int   = 250
	ttlUnixTimestamp int32 = 60 * 60 * 24 * 30
)
