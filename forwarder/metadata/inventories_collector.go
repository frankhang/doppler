package metadata

import (
	"expvar"
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/metadata/inventories"
	"github.com/frankhang/util/logutil"

	"github.com/frankhang/doppler/serializer"
	"github.com/frankhang/doppler/util"
)

type inventoriesCollector struct {
	ac   inventories.AutoConfigInterface
	coll inventories.CollectorInterface
	sc   *Scheduler
}

var (
	expvarPayload func() interface{}
)

func createPayload(ac inventories.AutoConfigInterface, coll inventories.CollectorInterface) (*inventories.Payload, error) {
	hostname, err := util.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("unable to submit inventories metadata payload, no hostname: %s", err)
	}

	return inventories.GetPayload(hostname, ac, coll), nil
}

// Send collects the data needed and submits the payload
func (c inventoriesCollector) Send(s *serializer.Serializer) error {
	if s == nil {
		return nil
	}

	payload, err := createPayload(c.ac, c.coll)
	if err != nil {
		return err
	}

	if err := s.SendMetadata(payload); err != nil {
		return fmt.Errorf("unable to submit inventories payload, %s", err)
	}
	return nil
}

// Init initializes the inventory metadata collection
func (c inventoriesCollector) Init() error {
	return inventories.StartMetadataUpdatedGoroutine(c.sc, config.Datadog.GetDuration("inventories_min_interval")*time.Second)
}

// SetupInventoriesExpvar init the expvar function for inventories
func SetupInventoriesExpvar(ac inventories.AutoConfigInterface, coll inventories.CollectorInterface) {
	expvar.Publish("inventories", expvar.Func(func() interface{} {
		logutil.BgLogger().Debug("Creating inventory payload for expvar")
		p, err := createPayload(ac, coll)
		if err != nil {
			logutil.BgLogger().Error("Could not create inventory payload for expvar", zap.Error(err))
			return &inventories.Payload{}
		}
		return p
	}))
}

// SetupInventories registers the inventories collector into the Scheduler and, if configured, schedules it
func SetupInventories(sc *Scheduler, ac inventories.AutoConfigInterface, coll inventories.CollectorInterface) error {
	ic := inventoriesCollector{
		ac:   ac,
		coll: coll,
		sc:   sc,
	}
	RegisterCollector("inventories", ic)

	if err := sc.AddCollector("inventories", config.Datadog.GetDuration("inventories_max_interval")*time.Second); err != nil {
		return err
	}

	SetupInventoriesExpvar(ac, coll)
	return nil
}
