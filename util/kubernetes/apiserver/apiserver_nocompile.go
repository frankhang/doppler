// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build !kubeapiserver

package apiserver

import (
	"errors"
	"fmt"
	"github.com/frankhang/util/logutil"

	apiv1 "github.com/frankhang/doppler/clusteragent/api/v1"
)

var (
	// ErrNotCompiled is returned if kubernetes apiserver support is not compiled in.
	// User classes should handle that case as gracefully as possible.
	ErrNotCompiled = errors.New("kubernetes apiserver support not compiled in")
)

// APIClient provides authenticated access to the
type APIClient struct {
	Cl interface{}
}

// metadataMapperBundle maps the podNames to the metadata they are associated with.
type metadataMapperBundle struct{}

// GetAPIClient returns the shared ApiClient instance.
func GetAPIClient() (*APIClient, error) {
	logutil.BgLogger().Error(fmt.Sprintf("GetAPIClient not implemented %s", ErrNotCompiled.Error()))
	return &APIClient{}, nil
}

// GetPodMetadataNames is used when the API endpoint of the DCA to get the services of a pod is hit.
func GetPodMetadataNames(nodeName, ns, podName string) ([]string, error) {
	logutil.BgLogger().Error(fmt.Sprintf("GetPodMetadataNames not implemented %s", ErrNotCompiled.Error()))
	return nil, nil
}

// GetMetadataMapBundleOnNode is used for the CLI svcmap command to output given a nodeName
func GetMetadataMapBundleOnNode(nodeName string) (*apiv1.MetadataResponse, error) {
	logutil.BgLogger().Error(fmt.Sprintf("GetMetadataMapBundleOnNode not implemented %s", ErrNotCompiled.Error()))
	return nil, nil
}

// GetMetadataMapBundleOnAllNodes is used for the CLI svcmap command to run fetch the service map of all nodes.
func GetMetadataMapBundleOnAllNodes(_ *APIClient) (*apiv1.MetadataResponse, error) {
	logutil.BgLogger().Error(fmt.Sprintf("GetMetadataMapBundleOnAllNodes not implemented %s", ErrNotCompiled.Error()))
	return nil, nil
}

// GetNodeLabels retrieves the labels of the queried node from the cache of the shared informer.
func GetNodeLabels(nodeName string) (map[string]string, error) {
	logutil.BgLogger().Error(fmt.Sprintf("GetNodeLabels not implemented %s", ErrNotCompiled.Error()))
	return nil, nil
}
