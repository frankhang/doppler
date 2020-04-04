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

	statsd.Namespace = "derun_"


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

		eRate := 0.1
		eTag := "error:1"
		var tags []string
		for i:=0; i<10;i++ {

			statsd.Tags = []string{"module:UserCenter", "env:dev", "role:provider"}
			if errorRate(eRate) {
				statsd.Tags = append(statsd.Tags, eTag)
			}
			tags = []string{"Method:GET", "Path:/api/of/UserCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"Method:POST", "Path:/api/of/UserCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:OrgCenter", "env:dev", "role:provider"}
			if errorRate(eRate) {
				statsd.Tags = append(statsd.Tags, eTag)
			}
			tags = []string{"Method:GET", "Path:/api/of/OrgCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"Method:POST", "Path:/api/of/OrgCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:DeviceCenter", "env:dev", "role:provider"}
			if errorRate(eRate) {
				statsd.Tags = append(statsd.Tags, eTag)
			}
			tags = []string{"Method:GET", "Path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"Method:POST", "Path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:TaskCenter", "env:dev", "role:provider"}
			if errorRate(eRate) {
				statsd.Tags = append(statsd.Tags, eTag)
			}
			tags = []string{"Method:GET", "Path:/api/of/TaskCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"Method:POST", "Path:/api/of/TaskCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:PolutionPlatform", "env:dev", "role:provider"}
			if errorRate(eRate) {
				statsd.Tags = append(statsd.Tags, eTag)
			}
			tags = []string{"Method:GET", "Path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"Method:POST", "Path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"ds:mysql1", "env:dev", "role:provider"}
			if errorRate(eRate) {
				statsd.Tags = append(statsd.Tags, eTag)

			}
			tags = []string{"Method:GET", "Path:/api/of/Other/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"Method:POST", "Path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

		}

		statsd.Tags = []string{"module:PolutionPlatform", "env:dev", "role:consumer"}
		if errorRate(eRate) {
			statsd.Tags = append(statsd.Tags, eTag)
		}
		tags = []string{"Method:GET", "Path:/api/of/UserCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)
		tags = []string{"Method:POST", "Path:/api/of/OrgCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)
		tags = []string{"Method:GET", "Path:/api/of/DeviceCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)
		tags = []string{"Method:POST", "Path:/api/of/TaskCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)




		err = statsd.Flush()
		errors.MustNil(err)
		time.Sleep(time.Second)
	}

}

func errorRate (rate float64) bool {

	if rand.Float64() < rate {
		return true
	}
	return false

}
