module github.com/frankhang/doppler

go 1.13

//replace github.com/frankhang/util => /Users/hang/go/src/github.com/frankhang/util

//replace github.com/frankhang/doppler => /Users/hang/go/src/github.com/frankhang/doppler

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/DataDog/agent-payload v0.0.0-20200317131523-f199c0eba1eb
	github.com/DataDog/datadog-agent v0.0.0-20200326104010-a1be18fb081f
	github.com/DataDog/datadog-go v3.3.1+incompatible
	github.com/DataDog/gohai v0.0.0-20200124154531-8cbe900337f1
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575
	github.com/dustin/go-humanize v1.0.0
	github.com/frankhang/util v0.0.0-20200326101710-e991a36b1b90
	github.com/goburrow/cache v0.1.0
	github.com/gogo/protobuf v1.2.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/json-iterator/go v1.1.8
	github.com/opentracing/opentracing-go v1.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.3.0
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	github.com/struCoder/pidusage v0.1.3
	github.com/twmb/murmur3 v1.1.2
	go.uber.org/automaxprocs v1.2.0
	go.uber.org/zap v1.13.0
	gopkg.in/yaml.v2 v2.2.4
)
