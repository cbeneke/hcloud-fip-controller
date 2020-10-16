package configuration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// VarsFromFile reads given config file and overwrite options from given Configuration
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

// Validate config options. Returns all errors found in a joined string
func (config *Configuration) Validate() error {
	var errs []string
	var undefinedErrs []string

	if config.HcloudAPIToken == "" {
		undefinedErrs = append(errs, "hetzner cloud API token")
	}
	if config.NodeName == "" {
		undefinedErrs = append(errs, "kubernetes node name")
	}
	if config.Namespace == "" {
		undefinedErrs = append(errs, "kubernetes namespace")
	}

	if config.LeaseDuration <= 0 {
		errs = append(errs, "lease duration needs to be greater than 0")
	}
	if config.LeaseRenewDeadline <= 0 {
		errs = append(errs, "lease renew deadline needs to be greater than 0")
	}
	if config.LeaseRenewDeadline >= config.LeaseDuration {
		errs = append(errs, "lease renew deadline needs to be smaller than lease duration")
	}

	if config.BackoffDuration == 0 {
		errs = append(errs, "backoff duration is not a valid duration or 0")
	}

	if config.BackoffFactor < 1 {
		errs = append(errs, "backoff factor must be at least 1")
	}

	if config.BackoffSteps < 0 {
		errs = append(errs, "backoff steps need to be greater than 0")
	}

	if len(undefinedErrs) > 0 {
		errs = append(errs, fmt.Sprintf("required configuration options not configured: %s", strings.Join(undefinedErrs, ", ")))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}
	return nil
}
