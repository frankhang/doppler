package main

import (
	"flag"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/frankhang/util/errors"
	"math/rand"
	"strconv"
	"time"
)

var (
	//bufio.NewWriterSize(p.BufReadConn, defaultWriterSize)
	url = flag.String("url", "localhost:8125", "host:port")
)

func main() {
	flag.Parse()

	println("connecting : " + *url)

	statsd, err := statsd.New(*url)
	errors.MustNil(err)

	defer statsd.Close()

	statsd.Namespace = "drht_"


	for {

		err := statsd.Count("example_metric_Count", 11, nil, 1)
		errors.MustNil(err)

		err = statsd.Gauge("example_metric_Gauge", 22, nil, 1)
		errors.MustNil(err)

		//err = statsd.Histogram("example_metric_Histogram", 33.33, []string{"Method:GET"}, 1)
		//errors.MustNil(err)

		//
		//err = statsd.Distribution("example_metric_Distribution", 44.44, nil, 1)
		//errors.MustNil(err)
		//
		//err = statsd.TimeInMilliseconds("example_metric_TimeInMilliseconds", 50, nil, 1)
		//errors.MustNil(err)
		//
		//err = statsd.Timing("example_metric_Timing", 222222, nil, 1)
		//errors.MustNil(err)

		//err = statsd.Set("example_metric_Set", "7777", nil, 1)
		//errors.MustNil(err)

		for i:=0; i<10;i++ {

			statsd.Tags = []string{"App:UserCenter", "Env:dev"}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:GET", "Path:/api/of/UserCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:POST", "Path:/api/of/UserCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"App:OrgCenter", "Env:dev"}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:GET", "Path:/api/of/OrgCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:POST", "Path:/api/of/OrgCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"App:DeviceCenter", "Env:dev"}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:GET", "Path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:POST", "Path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"App:TaskCenter", "Env:dev"}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:GET", "Path:/api/of/TaskCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), []string{"Method:POST", "Path:/api/of/TaskCenter/f" + strconv.Itoa(i)}, 1)
			errors.MustNil(err)


		}


		err = statsd.Flush()
		errors.MustNil(err)
		time.Sleep(time.Second)
	}

}
