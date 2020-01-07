package main

import (
	"context"
	"fmt"
	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/hack"
	"github.com/frankhang/util/logutil"
	"github.com/frankhang/util/tcp"
	l "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
)

const (
	sizeHead = 27
	locSize  = 18
)

//tierPacketIO implements PacketReader and PacketWriter
type PacketIO struct {
	*tcp.PacketIO

	driver *Driver
}

func NewPacketIO(packetIO *tcp.PacketIO, driver *Driver) *PacketIO {
	return &PacketIO{
		PacketIO: packetIO,
		driver:   driver,
	}
}

func (p *PacketIO) ReadPacket(ctx context.Context) (header []byte, data []byte, err error) {

	var size int

	header = p.Alloc.AllocWithLen(sizeHead, sizeHead)
	p.SetReadTimeout()
	if _, err = io.ReadFull(p.BufReadConn, header[:]); err != nil {
		return nil, nil, errors.Trace(err)
	}

	if l.GetLevel() >= l.DebugLevel {
		logutil.Logger(ctx).Debug("ReadPacket",
			zap.String("header", fmt.Sprintf("%x", header)),
			zap.String("headerStr", fmt.Sprintf("%s", header)),
		)

	}

	s := hack.String(header[locSize : locSize+3])
	if size, err = strconv.Atoi(strings.TrimSpace(s)); err != nil {
		return nil, nil, errors.Trace(err)
	}

	if l.GetLevel() >= l.DebugLevel {
		logutil.Logger(ctx).Debug("ReadPacket",
			zap.Int("size", size),
		)

	}

	if size > sizeHead {
		data = p.Alloc.AllocWithLen(size-sizeHead, size-sizeHead)

		p.SetReadTimeout()
		if _, err = io.ReadFull(p.BufReadConn, data); err != nil {
			return nil, nil, errors.Trace(err)
		}

	}

	return
}
