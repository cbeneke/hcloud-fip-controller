package configuration

import (
	"fmt"
	"testing"
	"time"
)

func testConfig() *Configuration {
	return &Configuration{
		HcloudAPIToken:     "token",
		HcloudFloatingIPs:  []string{"1.2.3.4"},
		LeaseDuration:      15,
		LeaseRenewDeadline: 10,
		LeaseName:          "fip",
		Namespace:          "fip",
		NodeAddressType:    "",
		NodeName:           "example",
		PodName:            "example",
		LogLevel:           "Info",
		BackoffDuration:    time.Second,
		BackoffFactor:      1.2,
		BackoffSteps:       5,
	}
}

var errorPrefix = "required configuration options not configured: "

type GenConfiguration func() *Configuration

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		config GenConfiguration
		err    error
	}{
		{
			name: "test valid config",
			config: func() *Configuration {
				return testConfig()
			},
			err: nil,
		},
		{
			name: "test no token",
			config: func() *Configuration {
				conf := testConfig()
				conf.HcloudAPIToken = ""
				return conf
			},
			err: fmt.Errorf(errorPrefix + "hetzner cloud API token"),
		},
		{
			name: "test no node name",
			config: func() *Configuration {
				conf := testConfig()
				conf.NodeName = ""
				return conf
			},
			err: fmt.Errorf(errorPrefix + "kubernetes node name"),
		},
		{
			name: "test no namespace",
			config: func() *Configuration {
				conf := testConfig()
				conf.Namespace = ""
				return conf
			},
			err: fmt.Errorf(errorPrefix + "kubernetes namespace"),
		},
		{
			name: "test lease duration too small and smaller the deadline",
			config: func() *Configuration {
				conf := testConfig()
				conf.LeaseDuration = 0
				return conf
			},
			err: fmt.Errorf("lease duration needs to be greater than 0, lease renew deadline needs to be smaller than lease duration"),
		},
		{
			name: "test lease deadline too small",
			config: func() *Configuration {
				conf := testConfig()
				conf.LeaseRenewDeadline = 0
				return conf
			},
			err: fmt.Errorf("lease renew deadline needs to be greater than 0"),
		},
		{
			name: "test backoff duration invalid",
			config: func() *Configuration {
				conf := testConfig()
				conf.BackoffDuration = 0
				return conf
			},
			err: fmt.Errorf("backoff duration is not a valid duration or 0"),
		},
		{
			name: "test backoff factor invalid",
			config: func() *Configuration {
				conf := testConfig()
				conf.BackoffFactor = 0.5
				return conf
			},
			err: fmt.Errorf("backoff factor must be at least 1"),
		},
		{
			name: "test backoff steps invalid",
			config: func() *Configuration {
				conf := testConfig()
				conf.BackoffSteps = -1
				return conf
			},
			err: fmt.Errorf("backoff steps need to be greater than 0"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conf := test.config()
			err := conf.Validate()

			if err == nil {
				if test.err != nil {
					t.Fatalf("error should be [%v] but was [nil]", test.err)
				}
			} else {
				if test.err == nil {
					t.Fatalf("error should be [nil] but was [%v]", err)
				}
				if err.Error() != test.err.Error() {
					t.Fatalf("error should be [%v] but was [%v]", test.err, err)
				}
			}
		})
	}
}
