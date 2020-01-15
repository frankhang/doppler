package main

import (
	"bufio"
	"context"
	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/tcp"
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


func (p *PacketIO) ReadPacket(ctx context.Context) ([]byte, []byte, error) {

	var frag []byte
	var full [][]byte
	var err error

	var delim byte = '\n'


	bufReader := p.BufReadConn.BufReader
	for {
		var e error
		p.SetReadTimeout()
		frag, e = bufReader.ReadSlice(delim)
		if e == nil { // got final fragment
			break
		}
		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}

		// Make a copy of the buffer.
		//l := len(frag)
		//buf := p.Alloc.AllocWithLen(l, l)
		//copy(buf, frag)
		//full = append(full, buf)
		full = append(full, frag)
	}

	// Allocate new buffer to hold the full pieces and the fragment.
	n := 0
	for i := range full {
		n += len(full[i])
	}
	n += len(frag)

	// Copy full pieces and fragment in.
	//buf := make([]byte, n)
	buf := p.Alloc.AllocWithLen(n, n)
	n = 0
	for i := range full {
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag)

	return nil, buf, errors.Trace(err)

}
