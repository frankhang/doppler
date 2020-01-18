// +build kubelet

package util

import (
	"github.com/frankhang/doppler/util/kubernetes/kubelet"
)

func isAgentKubeHostNetwork() (bool, error) {
	ku, err := kubelet.GetKubeUtil()
	if err != nil {
		return true, err
	}

	return ku.IsAgentHostNetwork()
}
