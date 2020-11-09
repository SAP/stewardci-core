package cfg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/featureflag"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	pipelineRunsConfigMapName            = "steward-pipelineruns"
	pipelineRunsConfigKeyTimeout         = "timeout"
	pipelineRunsConfigKeyLimitRange      = "limitRange"
	pipelineRunsConfigKeyResourceQuota   = "resourceQuota"
	pipelineRunsConfigKeyImage           = "jenkinsfileRunner.image"
	pipelineRunsConfigKeyImagePullPolicy = "jenkinsfileRunner.imagePullPolicy"
	pipelineRunsConfigKeyPSCRunAsUser    = "jenkinsfileRunner.podSecurityContext.runAsUser"
	pipelineRunsConfigKeyPSCRunAsGroup   = "jenkinsfileRunner.podSecurityContext.runAsGroup"
	pipelineRunsConfigKeyPSCFSGroup      = "jenkinsfileRunner.podSecurityContext.fsGroup"

	networkPoliciesConfigMapName    = "steward-pipelineruns-network-policies"
	networkPoliciesConfigKeyDefault = "_default"
)

// PipelineRunsConfigStruct is a struct holding the pipeline runs configuration
type PipelineRunsConfigStruct struct {
	Timeout                                       *metav1.Duration
	NetworkPolicies                               map[string]string
	DefaultNetworkPolicy                          string
	LimitRange                                    string
	ResourceQuota                                 string
	JenkinsfileRunnerImage                        string
	JenkinsfileRunnerImagePullPolicy              string
	JenkinsfileRunnerPodSecurityContextRunAsUser  *int64
	JenkinsfileRunnerPodSecurityContextRunAsGroup *int64
	JenkinsfileRunnerPodSecurityContextFSGroup    *int64
}

// LoadPipelineRunsConfig loads the pipelineruns configuration and returns it.
func LoadPipelineRunsConfig(clientFactory k8s.ClientFactory) (*PipelineRunsConfigStruct, error) {
	configMapIfce := clientFactory.CoreV1().ConfigMaps(system.Namespace())
	config := &PipelineRunsConfigStruct{}

	configMap, err := configMapIfce.Get(pipelineRunsConfigMapName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, withRecoverability(err, true)
	}
	if configMap != nil {
		err = processMainConfig(configMap.Data, config)
		if err != nil {
			return nil, withRecoverability(err, false)
		}
	}

	networkMap, err := configMapIfce.Get(networkPoliciesConfigMapName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, withRecoverability(err, true)
	}

	if networkMap != nil {
		err = processNetworkMap(networkMap.Data, config)
		if err != nil {
			return nil, withRecoverability(err, false)
		}
	}
	return config, nil
}

func withRecoverability(err error, isInfraError bool) error {
	return serrors.RecoverableIf(err, isInfraError || featureflag.RetryOnInvalidPipelineRunsConfig.Enabled())
}

func processMainConfig(configData map[string]string, config *PipelineRunsConfigStruct) error {
	config.LimitRange = configData[pipelineRunsConfigKeyLimitRange]
	config.ResourceQuota = configData[pipelineRunsConfigKeyResourceQuota]
	config.JenkinsfileRunnerImage = configData[pipelineRunsConfigKeyImage]
	config.JenkinsfileRunnerImagePullPolicy = configData[pipelineRunsConfigKeyImagePullPolicy]

	parseInt64 := func(key string) (*int64, error) {
		if strVal, ok := configData[key]; ok && strVal != "" {
			intVal, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot parse configuration value %q at %q", strVal, key)
			}
			return &intVal, nil
		}
		return nil, nil
	}

	parseDuration := func(key string) (*metav1.Duration, error) {
		if strVal, ok := configData[key]; ok && strVal != "" {
			d, err := time.ParseDuration(strVal)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot parse configuration value %q at %q", strVal, key)
			}
			return &metav1.Duration{Duration: d}, nil
		}
		return nil, nil
	}
	var err error
	if config.Timeout, err =
		parseDuration(pipelineRunsConfigKeyTimeout); err != nil {
		return err
	}

	if config.JenkinsfileRunnerPodSecurityContextRunAsUser, err =
		parseInt64(pipelineRunsConfigKeyPSCRunAsUser); err != nil {
		return err
	}
	if config.JenkinsfileRunnerPodSecurityContextRunAsGroup, err =
		parseInt64(pipelineRunsConfigKeyPSCRunAsGroup); err != nil {
		return err
	}

	if config.JenkinsfileRunnerPodSecurityContextFSGroup, err =
		parseInt64(pipelineRunsConfigKeyPSCFSGroup); err != nil {
		return err
	}

	return nil
}

func processNetworkMap(networkMap map[string]string, config *PipelineRunsConfigStruct) error {
	isValidKey := func(key string) bool {
		trimmedKey := strings.TrimSpace(key)
		return trimmedKey != "" && key == trimmedKey && !strings.HasPrefix(trimmedKey, "_")
	}
	defaultNetworkPolicyKey := networkMap[networkPoliciesConfigKeyDefault]
	if !isValidKey(defaultNetworkPolicyKey) {
		return fmt.Errorf(
			"invalid configuration: ConfigMap %q in namespace %q: key %q is missing or invalid",
			pipelineRunsConfigMapName,
			system.Namespace(),
			networkPoliciesConfigKeyDefault,
		)
	}
	if config.DefaultNetworkPolicy = strings.TrimSpace(networkMap[defaultNetworkPolicyKey]); config.DefaultNetworkPolicy == "" {
		return fmt.Errorf(
			"invalid configuration: ConfigMap %q in namespace %q: key %q: "+
				"no network policy with key %q found",
			pipelineRunsConfigMapName,
			system.Namespace(),
			networkPoliciesConfigKeyDefault,
			defaultNetworkPolicyKey,
		)
	}

	networkPolicies := map[string]string{}
	for key, value := range networkMap {
		if key != defaultNetworkPolicyKey && isValidKey(key) && strings.TrimSpace(value) != "" {
			networkPolicies[key] = value
		}
	}
	if len(networkPolicies) != 0 {
		config.NetworkPolicies = networkPolicies
	}
	return nil
}
