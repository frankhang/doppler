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

		err := statsd.Count("example_metric.Count", 11, []string{"environment:dev"}, 1)
		errors.MustNil(err)

		err = statsd.Gauge("example_metric.Gauge", 22, []string{"environment:prod"}, 1)
		errors.MustNil(err)

		err = statsd.Histogram("example_metric.Histogram", 33.33, []string{"environment:prod"}, 1)
		errors.MustNil(err)

		err = statsd.Distribution("example_metric.Distribution", 44.44, []string{"environment:prod"}, 1)
		errors.MustNil(err)

		err = statsd.TimeInMilliseconds("example_metric.TimeInMilliseconds", 55.55, []string{"environment:prod"}, 1)
		errors.MustNil(err)

		err = statsd.Timing("example_metric.Timing", 6666, []string{"environment:prod"}, 1)
		errors.MustNil(err)


		err = statsd.Flush()
		errors.MustNil(err)
		time.Sleep(time.Second)
	}


}
