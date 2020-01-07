package main

import (
	"context"
	"fmt"
	"github.com/frankhang/util/logutil"
	"github.com/frankhang/util/tcp"
	"github.com/frankhang/util/util"
	l "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

//tierHandler implements Hanlder
type Handler struct {
	*PacketIO

	driver *Driver

	ctl *Controller
}

func NewHandler(tierPacketIO *PacketIO, driver *Driver) *Handler {
	handler := &Handler{PacketIO: tierPacketIO, driver: driver}
	handler.ctl = &Controller{PacketIO: tierPacketIO}
	return handler
}

func (th *Handler) Handle(ctx context.Context, cc *tcp.ClientConn, header []byte, data []byte) error {

	ctl := th.ctl
	ctl.cc = cc

	sum := util.Sum(header)
	ssum := util.SignedSum(header)

	if len(data) >= 3 {
		sum += util.Sum(data[:len(data)-3])
		ssum += util.SignedSum(data[:len(data)-3])
	}

	//ctx = logutil.WithInt(ctx, "length", len(header)+len(data))
	//ctx = logutil.WithString(ctx, "packet", fmt.Sprintf("%x%x", header, data))
	//ctx = logutil.WithInt(ctx, "sum", sum)

	if l.GetLevel() >= l.DebugLevel {
		logutil.Logger(ctx).Debug("Packet received",
			zap.Int("size", len(header)+len(data)),
			zap.String("packet", fmt.Sprintf("%x%x", header, data)),
			zap.Int("sum", sum),
			zap.Int("ssum", ssum),
			zap.String("packetStr", fmt.Sprintf("%s%s", header, data)),
		)
	}

	//cmd := hack.String(header[:2])
	////dispach cmd process logic to controller
	//switch cmd {
	//case "55":
	//	ctl.ctx = logutil.WithString(ctx, "method", "TirePressureReport")
	//	err := ctl.TirePressureReport(header, data)
	//	return errors.Trace(err)
	//case "57":
	//	ctl.ctx = logutil.WithString(ctx, "method", "TireReplaceAck")
	//	err := ctl.TireReplaceAck(header, data)
	//	return errors.Trace(err)
	//}

	logutil.Logger(ctx).Warn("no controller method found")
	return nil
}
