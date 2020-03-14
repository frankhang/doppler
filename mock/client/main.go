package main

import (
	"flag"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/frankhang/util/errors"
	"time"
)

var (
	//bufio.NewWriterSize(p.BufReadConn, defaultWriterSize)
	url   = flag.String("url", "localhost:8125", "host:port")
)

func main() {
	flag.Parse()

	println("connecting : " + *url)

	statsd, err := statsd.New(*url)
	errors.MustNil(err)

	defer statsd.Close()

	for {

		err := statsd.Count("example_metric.count", 2, []string{"environment:dev"}, 1)
		errors.MustNil(err)

		err = statsd.Flush()

		err = statsd.Gauge("example_metric.gauge", 10, []string{"environment:prod"}, 1)
		errors.MustNil(err)

		err = statsd.Flush()
		errors.MustNil(err)
		time.Sleep(time.Second)
	}


}
