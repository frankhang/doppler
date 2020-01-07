package main

import (
	"crypto/tls"
	"github.com/frankhang/util/config"
	"github.com/frankhang/util/tcp"
)

// Driver implements tcp.IDriver.
type Driver struct {
	cfg *config.Config
}

// NewTireDriver creates a new Driver.
func NewTireDriver(cfg *config.Config) *Driver {
	driver := &Driver{
		cfg: cfg,
	}

	return driver
}

// TireContext implements QueryCtx.
type TireContext struct {
	currentDB string
}

// TireStatement implements PreparedStatement.
type TireStatement struct {
	id  uint32
	ctx *TireContext
}

func (td *Driver) OpenCtx(connID uint64, capability uint32, collation uint8, dbname string, tlsState *tls.ConnectionState) (tcp.QueryCtx, error) {
	return nil, nil
}

func (td *Driver) GeneratePacketIO(cc *tcp.ClientConn) *tcp.PacketIO {
	packetIO := tcp.NewPacketIO(cc)

	tierPacketIO := NewPacketIO(packetIO, td)
	tierHandler := NewHandler(tierPacketIO, td)

	packetIO.PacketReader = tierPacketIO
	packetIO.PacketWriter = tierPacketIO

	cc.Handler = tierHandler

	return packetIO
}
