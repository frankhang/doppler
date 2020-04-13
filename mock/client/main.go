package main

import (
	"flag"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/frankhang/util/errors"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const (
	e400Tag    = "code:400"
	e401Tag    = "code:401"
	rTag       = "code:200"
	timeoutTag = "code:timeout"
)

var (
	//bufio.NewWriterSize(p.BufReadConn, defaultWriterSize)
	url  = flag.String("url", "localhost:8125", "host:port")
	host string
)

func main() {
	flag.Parse()

	println("connecting : " + *url)

	dClient, err := statsd.New(*url)
	errors.MustNil(err)

	defer dClient.Close()

	host, err = os.Hostname()
	errors.MustNil(err)

	dClient.Namespace = "derun_"

	for {

		//err := dClient.Count("example_metric_Count", 11, nil, 1)
		//errors.MustNil(err)
		//
		//err = dClient.Gauge("example_metric_Gauge", 22, nil, 1)
		//errors.MustNil(err)

		//err = dClient.Histogram("example_metric_Histogram", 33.33, []string{"method:GET"}, 1)
		//errors.MustNil(err)

		//
		//err = dClient.Distribution("example_metric_Distribution", 44.44, nil, 1)
		//errors.MustNil(err)
		//
		//err = dClient.TimeInMilliseconds("example_metric_TimeInMilliseconds", 50, nil, 1)
		//errors.MustNil(err)
		//
		//err = dClient.Timing("example_metric_Timing", 222222, nil, 1)
		//errors.MustNil(err)

		//err = dClient.Set("example_metric_Set", "7777", nil, 1)
		//errors.MustNil(err)

		var tags []string
		for i := 0; i < 5; i++ {

			if hitRate((float64((i)+1) / 5)) {

				println("send" + strconv.Itoa(i))
				dClient.Tags = []string{"module:UserCenter", "env:dev", "role:provider"}
				dClient.Tags = appendCodeTag(dClient.Tags)

				tags = []string{"method:GET", "path:/api/of/UserCenter/f" + strconv.Itoa(i)}
				err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				errors.MustNil(err)

				if hitRate(0.5) {
					tags = []string{"method:POST", "path:/api/of/UserCenter/f" + strconv.Itoa(i)}
					err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
					errors.MustNil(err)
				}
				sc := statsd.NewServiceCheck("derun_UserCenter", statsd.Ok)
				sc.Hostname = host
				sc.Tags = []string{"env:dev"}
				dClient.ServiceCheck(sc)



				dClient.Tags = []string{"module:OrgCenter", "env:dev", "role:provider"}
				dClient.Tags = appendCodeTag(dClient.Tags)
				tags = []string{"method:GET", "path:/api/of/OrgCenter/f" + strconv.Itoa(i)}
				err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				errors.MustNil(err)

				if hitRate(0.5) {
					tags = []string{"method:POST", "path:/api/of/OrgCenter/f" + strconv.Itoa(i)}
					err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
					errors.MustNil(err)
				}
				sc = statsd.NewServiceCheck("derun_OrgCenter", statsd.Warn)
				sc.Hostname = host
				sc.Tags = []string{"env:dev"}
				dClient.ServiceCheck(sc)




				dClient.Tags = []string{"module:DeviceCenter", "env:dev", "role:provider"}
				dClient.Tags = appendCodeTag(dClient.Tags)
				tags = []string{"method:GET", "path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}
				err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				errors.MustNil(err)

				if hitRate(0.5) {

					tags = []string{"method:POST", "path:/api/of/DeviceCenter/f" + strconv.Itoa(i)}
					err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
					errors.MustNil(err)
				}
				sc = statsd.NewServiceCheck("derun_DeviceCenter", statsd.Critical)
				sc.Hostname = host
				sc.Tags = []string{"env:dev"}
				dClient.ServiceCheck(sc)



				dClient.Tags = []string{"module:TaskCenter", "env:dev", "role:provider"}
				dClient.Tags = appendCodeTag(dClient.Tags)
				tags = []string{"method:GET", "path:/api/of/TaskCenter/f" + strconv.Itoa(i)}
				err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				errors.MustNil(err)

				if hitRate(0.5) {

					tags = []string{"method:POST", "path:/api/of/TaskCenter/f" + strconv.Itoa(i)}
					err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
					errors.MustNil(err)
				}
				sc = statsd.NewServiceCheck("TaskCenter", statsd.Unknown)
				sc.Hostname = host
				sc.Tags = []string{"env:dev"}
				dClient.ServiceCheck(sc)


				dClient.Tags = []string{"module:PolutionPlatform", "env:dev", "role:provider"}
				dClient.Tags = appendCodeTag(dClient.Tags)
				tags = []string{"method:GET", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
				err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				errors.MustNil(err)

				if hitRate(0.5) {

					tags = []string{"method:POST", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
					err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
					errors.MustNil(err)
				}
				//sc = statsd.NewServiceCheck("derun_PolutionPlatform", statsd.Ok)
				//sc.Hostname = host
				//sc.Tags = []string{"env:dev"}
				//dClient.ServiceCheck(sc)


				//dClient.Tags = []string{"module:PolutionPlatform", "env:dev", "role:provider"}
				//dClient.Tags = appendCodeTag(dClient.Tags)
				//
				//tags = []string{"method:GET", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
				//err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				//errors.MustNil(err)
				//
				//if hitRate(0.5) {
				//
				//	tags = []string{"method:POST", "path:/api/of/PolutionPlatform/f" + strconv.Itoa(i)}
				//	err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
				//	errors.MustNil(err)
				//}
				//sc = statsd.NewServiceCheck("derun_PolutionPlatform", statsd.Ok)
				//sc.Hostname = host
				//sc.Tags = []string{"env:dev"}
				//dClient.ServiceCheck(sc)
			}

		}

		println("send consumer")
		dClient.Tags = []string{"module:PolutionPlatform", "env:dev", "role:consumer"}
		dClient.Tags = appendCodeTag(dClient.Tags)

		if hitRate(0.8) {

			tags = []string{"method:GET", "path:/api/of/UserCenter/f1"}
			err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
		}

		if hitRate(0.7) {

			tags = []string{"method:POST", "path:/api/of/OrgCenter/f1"}
			err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
		}
		if hitRate(0.6) {

			tags = []string{"method:GET", "path:/api/of/DeviceCenter/f1"}
			err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
		}
		if hitRate(0.5) {

			tags = []string{"method:POST", "path:/api/of/TaskCenter/f1"}
			err = dClient.Histogram("api", float64(int(rand.Float64()*1000)), tags, 1)
			errors.MustNil(err)
		}

		err = dClient.Flush()
		errors.MustNil(err)

		time.Sleep(time.Millisecond * 200)

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

	newTags = append(newTags, fmt.Sprintf("hostname:%s", host))
	return

}

func hitRate(rate float64) bool {
	r := rand.Float64()
	if r < rate {
		return true
	}
	return false
}
