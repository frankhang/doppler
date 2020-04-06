package main

import (
	"flag"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/frankhang/util/errors"
	"math/rand"
	"strconv"
	"time"
)

const (
	e400Tag = "code:400"
	e401Tag = "code:401"
	rTag = "code:200"
	timeoutTag = "code:timeout"
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

		//err := statsd.Count("example_metric_Count", 11, nil, 1)
		//errors.MustNil(err)
		//
		//err = statsd.Gauge("example_metric_Gauge", 22, nil, 1)
		//errors.MustNil(err)

		//err = statsd.Histogram("example_metric_Histogram", 33.33, []string{"method:GET"}, 1)
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

		var tags []string
		for i := 0; i < 10; i++ {

			statsd.Tags = []string{"module:UserCenter", "env:dev", "role:provider", "ds:"}
			statsd.Tags = appendCodeTag(statsd.Tags)

			tags = []string{"method:GET", "path:/api/of/UserCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"method:POST", "path:/api/of/UserCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:OrgCenter", "env:dev", "role:provider", "ds:"}
			statsd.Tags = appendCodeTag(statsd.Tags)
			tags = []string{"method:GET", "path:/api/of/OrgCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"method:POST", "path:/api/of/OrgCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:DeviceCenter", "env:dev", "role:provider", "ds:"}
			statsd.Tags = appendCodeTag(statsd.Tags)
			tags = []string{"method:GET", "path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"method:POST", "path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:TaskCenter", "env:dev", "role:provider", "ds:"}
			statsd.Tags = appendCodeTag(statsd.Tags)
			tags = []string{"method:GET", "path:/api/of/TaskCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"method:POST", "path:/api/of/TaskCenter/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:PolutionPlatform", "env:dev", "role:provider", "ds:"}
			statsd.Tags = appendCodeTag(statsd.Tags)
			tags = []string{"method:GET", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"method:POST", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

			statsd.Tags = []string{"module:PolutionPlatform", "env:dev", "role:provider", "ds:"}
			statsd.Tags = appendCodeTag(statsd.Tags)

			tags = []string{"method:GET", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
			tags = []string{"method:POST", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
			err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)

		}

		statsd.Tags = []string{"module:PolutionPlatform", "env:dev", "role:consumer", "ds:"}
		statsd.Tags = appendCodeTag(statsd.Tags)
		tags = []string{"method:GET", "path:/api/of/UserCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)
		tags = []string{"method:POST", "path:/api/of/OrgCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)
		tags = []string{"method:GET", "path:/api/of/DeviceCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)
		tags = []string{"method:POST", "path:/api/of/TaskCenter/f1"}
		err = statsd.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
		errors.MustNil(err)

		err = statsd.Flush()
		errors.MustNil(err)
		time.Sleep(time.Second)
	}

}

func appendCodeTag(tags []string) (newTags []string) {
	r := rand.Float64()
	if r < 0.05 {
		newTags = append(tags, timeoutTag)
	} else if r < 0.15 {
		newTags = append(tags, e400Tag)
	} else if r < 0.3 {
		newTags = append(tags, e401Tag)
	} else {
		newTags = append(tags, rTag)
	}
	return

}

