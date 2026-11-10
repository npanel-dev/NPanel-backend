package server

import "errors"

var (
	ErrProtocolTypeRequired  = errors.New("protocol type is required")
	ErrDuplicateProtocolType = errors.New("duplicate protocol type")
	ErrDuplicateProtocolPort = errors.New("duplicate protocol port")
	ErrInvalidProtocolConfig = errors.New("invalid protocol configuration")
	ErrSharedProtocolInstance = errors.New("protocol instance is shared by multiple nodes")
)
