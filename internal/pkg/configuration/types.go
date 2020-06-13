package configuration

import (
	"fmt"
	"strings"
)

// Configuration has all configurable values for the fip-controller
// All values can be configured via config file, cli params and envrionment variables
type Configuration struct {
	HcloudAPIToken          string           `json:"hcloud_api_token,omitempty"`
	HcloudFloatingIPs       stringArrayFlags `json:"hcloud_floating_ips,omitempty"`
	LeaseDuration           int              `json:"lease_duration,omitempty"`
	LeaseName               string           `json:"lease_name,omitempty"`
	Namespace               string           `json:"namespace,omitempty"`
	NodeAddressType         NodeAddressType  `json:"node_address_type,omitempty"`
	NodeLabelSelector       string           `json:"node_label_selector,omitempty"`
	PodLabelSelector        string           `json:"pod_label_selector,omitempty"`
	NodeName                string           `json:"node_name,omitempty"`
	PodName                 string           `json:"pod_name,omitempty"`
	LogLevel                string           `json:"log_level,omitempty"`
	FloatingIPLabelSelector string           `json:"floating_ip_label_selector,omitempty"`
}

// Set of string flags
type stringArrayFlags []string

func (flags *stringArrayFlags) String() string {
	return fmt.Sprintf("['%s']", strings.Join(*flags, "', '"))
}
func (flags *stringArrayFlags) Set(value string) error {
	*flags = append(*flags, value)
	return nil
}

// NodeAddressType specifies valid node address types
type NodeAddressType string

const (
	// NodeAddressTypeExternal is the constant for external node address types
	NodeAddressTypeExternal = "external"
	// NodeAddressTypeInternal is the constant for internal node address types
	NodeAddressTypeInternal = "internal"
)

func (flags *NodeAddressType) String() string {
	return string(*flags)
}

// Set is used for setting the node address type
// This function is required to satisfy the flag interface
func (flags *NodeAddressType) Set(value string) error {
	*flags = NodeAddressType(value)
	return nil
}
