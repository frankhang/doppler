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

	statsd.Namespace = "ns1_"
	statsd.Tags = []string{"App:UserCenter", "Env:dev"}


	for {

		err := statsd.Count("example_metric_Count", 11, nil, 1)
		errors.MustNil(err)

		err = statsd.Gauge("example_metric_Gauge", 22, nil, 1)
		errors.MustNil(err)

		err = statsd.Histogram("example_metric_Histogram", 33.33, []string{"Method:GET"}, 1)
		errors.MustNil(err)

		err = statsd.Distribution("example_metric_Distribution", 44.44, nil, 1)
		errors.MustNil(err)

		err = statsd.TimeInMilliseconds("example_metric_TimeInMilliseconds", 5555, nil, 1)
		errors.MustNil(err)

		err = statsd.Timing("example_metric_Timing", 222222, nil, 1)
		errors.MustNil(err)

		err = statsd.Set("example_metric._Set", "7777", nil, 1)
		errors.MustNil(err)


		err = statsd.Flush()
		errors.MustNil(err)
		time.Sleep(time.Millisecond*200)
	}


}
