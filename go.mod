module github.com/frankhang/iot

go 1.13

replace github.com/frankhang/util => /Users/hang/go/src/github.com/frankhang/util

replace github.com/frankhang/doppler => /Users/hang/go/src/github.com/frankhang/doppler

require (
	github.com/frankhang/util v0.0.0-00010101000000-000000000000
	github.com/opentracing/opentracing-go v1.1.0
	github.com/prometheus/client_golang v1.3.0
	github.com/sirupsen/logrus v1.4.2
	github.com/struCoder/pidusage v0.1.3
	go.uber.org/automaxprocs v1.2.0
	go.uber.org/zap v1.13.0
)
