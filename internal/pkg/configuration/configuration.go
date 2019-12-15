package configuration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/namsral/flag"
)

// Read given config file and overwrite options from given Configuration
func (config *Configuration) VarsFromFile(configFile string) error {
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config file: %v", err)
	}

	return nil
}

func (config *Configuration) Validate() error {
	var errs []string
	var undefinedErrs []string

	if config.HcloudApiToken == "" {
		undefinedErrs = append(errs, "hetzner cloud API token")
	}
	if len(config.HcloudFloatingIPs) <= 0 {
		undefinedErrs = append(errs, "hetzner cloud floating IPs")
	}
	if config.NodeName == "" {
		undefinedErrs = append(errs, "kubernetes node name")
	}
	if config.Namespace == "" {
		undefinedErrs = append(errs, "kubernetes namespace")
	}
	if config.LeaseDuration <= 0 {
		errs = append(errs, "lease duration needs to be greater than one")
	}

	if len(undefinedErrs) > 0 {
		errs = append(errs, fmt.Sprintf("required configuration options not configured: %s", strings.Join(errs, ", ")))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}
	return nil
}

func (config Configuration) ParseFlags() {
	// Setup flags
	flag.Var(&config.HcloudFloatingIPs, "hcloud-floating-ip", "Hetzner cloud floating IP Address. This option can be specified multiple times")

	flag.StringVar(&config.HcloudApiToken, "hcloud-api-token", "", "Hetzner cloud API token")
	flag.IntVar(&config.LeaseDuration, "lease-duration", 30, "Time to wait (in seconds) until next leader check")
	flag.StringVar(&config.LeaseName, "lease-name", "fip", "Name of the lease lock for leaderelection")
	flag.StringVar(&config.Namespace, "namespace", "", "Kubernetes Namespace")
	flag.StringVar(&config.NodeAddressType, "node-address-type", "external", "Kubernetes node address type")
	flag.StringVar(&config.NodeName, "node-name", "", "Kubernetes Node name")
	flag.StringVar(&config.PodName, "pod-name", "", "Kubernetes pod name")
	flag.StringVar(&config.LogLevel, "log-level", "Info", "Log level")

	// Parse options from file
	if _, err := os.Stat("config/config.json"); err == nil {
		if err := config.VarsFromFile("config/config.json"); err != nil {
			fmt.Println(fmt.Errorf("could not parse controller config file: %v", err))
			os.Exit(1)
		}
	}

	// When default- and file-configs are read, parse command line options with highest priority
	flag.Parse()
}
