package main

import (
	"context"
	"github.com/frankhang/util/logutil"
	"github.com/frankhang/util/tcp"
)

type Controller struct {
	*PacketIO

	ctx context.Context
	cc  *tcp.ClientConn
}

func (c *Controller) TirePressureReport(header []byte, data []byte) error {
	logutil.Logger(c.ctx).Info("controller")

	return nil

}

func (c *Controller) TireReplaceAck(header []byte, data []byte) error {
	logutil.Logger(c.ctx).Info("controller")
	return nil

}
