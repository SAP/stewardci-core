package custom

import (
	yaml "gopkg.in/yaml.v3"
)

// GetLoggingDetailsProvider parses the given YAML-formatted configuration string
// and returns a corresponding LoggingDetailsProvider.
// nil is returned if the configuration is empty or contains only entries that are
// not supported.
func GetLoggingDetailsProvider(configYAMLString string) (LoggingDetailsProvider, error) {
	var detailConfigs = []loggingDetailConfig{}
	if configYAMLString != "" {
		err := yaml.Unmarshal([]byte(configYAMLString), &detailConfigs)
		if err != nil {
			return nil, err
		}
	}

	if len(detailConfigs) == 0 {
		return nil, nil
	}

	providers := []LoggingDetailsProvider{}
	for _, detailConfig := range detailConfigs {
		constructorFunc := providerRegistry[detailConfig.Kind]
		if constructorFunc != nil {
			provider, err := constructorFunc(detailConfig.LogKey, detailConfig.Spec)
			if err != nil {
				return nil, err
			}
			if provider != nil {
				providers = append(providers, provider)
			}
		}
	}

	if len(providers) == 0 {
		return nil, nil
	}
	return mergeProviders(providers...), nil
}
