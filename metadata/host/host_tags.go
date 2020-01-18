package host

import (
	"fmt"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
	"strings"

	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/util/docker"
	//"github.com/frankhang/doppler/util/ec2"
	//"github.com/frankhang/doppler/util/gce"
	"github.com/frankhang/doppler/util/kubernetes/clustername"
	k8s "github.com/frankhang/doppler/util/kubernetes/hostinfo"
	//"github.com/frankhang/doppler/util/log"
)

// this is a "low-tech" version of tagger/utils/taglist.go
// but host tags are handled separately here for now
func appendAndSplitTags(target []string, tags []string, splits map[string]string) []string {
	if len(splits) == 0 {
		return append(target, tags...)
	}

	for _, tag := range tags {
		tagParts := strings.SplitN(tag, ":", 2)
		if len(tagParts) != 2 {
			target = append(target, tag)
			continue
		}
		name := tagParts[0]
		value := tagParts[1]

		sep, ok := splits[name]
		if !ok {
			target = append(target, tag)
			continue
		}

		for _, elt := range strings.Split(value, sep) {
			target = append(target, fmt.Sprintf("%s:%s", name, elt))
		}
	}

	return target
}

func getHostTags() *tags {
	splits := Cfg.TagValueSplitSeparator
	appendToHostTags := func(old, new []string) []string {
		return appendAndSplitTags(old, new, splits)
	}

	rawHostTags := Cfg.Tags
	hostTags := make([]string, 0, len(rawHostTags))
	hostTags = appendToHostTags(hostTags, rawHostTags)

	//if config.Datadog.GetBool("collect_ec2_tags") {
	//	ec2Tags, err := ec2.GetTags()
	//	if err != nil {
	//		log.Debugf("No EC2 host tags %v", err)
	//	} else {
	//		hostTags = appendToHostTags(hostTags, ec2Tags)
	//	}
	//}

	clusterName := clustername.GetClusterName()
	if len(clusterName) != 0 {
		hostTags = appendToHostTags(hostTags, []string{"cluster_name:" + clusterName})
	}

	k8sTags, err := k8s.GetTags()
	if err != nil {
		logutil.BgLogger().Info("No Kubernetes host tags", zap.Error(err))
	} else {
		hostTags = appendToHostTags(hostTags, k8sTags)
	}

	dockerTags, err := docker.GetTags()
	if err != nil {
		logutil.BgLogger().Info("No Docker host tags", zap.Error(err))
	} else {
		hostTags = appendToHostTags(hostTags, dockerTags)
	}

	//gceTags := []string{}
	//if config.Datadog.GetBool("collect_gce_tags") {
	//	rawGceTags, err := gce.GetTags()
	//	if err != nil {
	//		log.Debugf("No GCE host tags %v", err)
	//	} else {
	//		gceTags = appendToHostTags(gceTags, rawGceTags)
	//	}
	//}

	return &tags{
		System:              hostTags,
		//GoogleCloudPlatform: gceTags,
		GoogleCloudPlatform: hostTags,
	}
}
