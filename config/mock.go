// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package config



// MockConfig should only be used in tests
type MockConfig struct {
	Config
}

// Set is used for setting configuration in tests
func (c *MockConfig) Set(key string, value interface{}) {

}

// Mock is creating and returning a mock config
func Mock() *MockConfig {

	return nil
}
