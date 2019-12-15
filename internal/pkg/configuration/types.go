package configuration

import (
	"fmt"
	"strings"
)

type Configuration struct {
	HcloudApiToken    string           `json:"hcloud_api_token,omitempty"`
	HcloudFloatingIPs stringArrayFlags `json:"hcloud_floating_ips,omitempty"`
	LeaseDuration     int              `json:"lease_duration,omitempty"`
	LeaseName         string           `json:"lease_name,omitempty"`
	Namespace         string           `json:"namespace,omitempty"`
	NodeAddressType   string           `json:"node_address_type,omitempty"`
	NodeName          string           `json:"node_name,omitempty"`
	PodName           string           `json:"pod_name,omitempty"`
	LogLevel          string           `json:"log_level,omitempty"`
}

type stringArrayFlags []string

func (flags *stringArrayFlags) String() string {
	return fmt.Sprintf("['%s']", strings.Join(*flags, "', '"))
}

func (flags *stringArrayFlags) Set(value string) error {
	*flags = append(*flags, value)
	return nil
}
