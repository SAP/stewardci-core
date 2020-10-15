package runctl

import (
	"strconv"
	"strings"
	"time"

	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	pipelineRunsConfigMapName          = "steward-pipelineruns"
	pipelineRunsConfigKeyTimeout       = "timeout"
	pipelineRunsConfigKeyLimitRange    = "limitRange"
	pipelineRunsConfigKeyResourceQuota = "resourceQuota"
	pipelineRunsConfigKeyPSCRunAsUser  = "jenkinsfileRunner.podSecurityContext.runAsUser"
	pipelineRunsConfigKeyPSCRunAsGroup = "jenkinsfileRunner.podSecurityContext.runAsGroup"
	pipelineRunsConfigKeyPSCFSGroup    = "jenkinsfileRunner.podSecurityContext.fsGroup"

	networkPoliciesConfigMapName    = "steward-pipelineruns-network-policies"
	networkPoliciesConfigKeyDefault = "_default"
)

type pipelineRunsConfigStruct struct {
	Timeout                                       *metav1.Duration
	NetworkPolicies                               map[string]string
	DefaultNetworkPolicy                          string
	LimitRange                                    string
	ResourceQuota                                 string
	JenkinsfileRunnerPodSecurityContextRunAsUser  *int64
	JenkinsfileRunnerPodSecurityContextRunAsGroup *int64
	JenkinsfileRunnerPodSecurityContextFSGroup    *int64
}

func loadPipelineRunsConfig(clientFactory k8s.ClientFactory) (*pipelineRunsConfigStruct, error) {
	configMapIfce := clientFactory.CoreV1().ConfigMaps(system.Namespace())

	configMap, err := configMapIfce.Get(pipelineRunsConfigMapName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		return nil, serrors.Recoverable(err)
	}

	networkMap, err := configMapIfce.Get(networkPoliciesConfigMapName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		return nil, serrors.Recoverable(err)
	}

	defaultNetworkPolicyKey := networkMap.Data[networkPoliciesConfigKeyDefault]
	networkPolicies := map[string]string{}
	for key, value := range networkMap.Data {
		if key != defaultNetworkPolicyKey && !strings.HasPrefix(key, "_") {
			networkPolicies[key] = value
		}
	}
	if len(networkPolicies) == 0 {
		networkPolicies = nil
	}

	config := &pipelineRunsConfigStruct{
		LimitRange:           configMap.Data[pipelineRunsConfigKeyLimitRange],
		ResourceQuota:        configMap.Data[pipelineRunsConfigKeyResourceQuota],
		NetworkPolicies:      networkPolicies,
		DefaultNetworkPolicy: networkMap.Data[defaultNetworkPolicyKey],
	}

	parseInt64 := func(key string) (*int64, error) {
		if strVal, ok := configMap.Data[key]; ok && strVal != "" {
			intVal, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot parse configuration value %q at %q", strVal, key)
			}
			return &intVal, nil
		}
		return nil, nil
	}

	parseDuration := func(key string) (*metav1.Duration, error) {
		if strVal, ok := configMap.Data[key]; ok && strVal != "" {
			d, err := time.ParseDuration(strVal)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot parse configuration value %q at %q", strVal, key)
			}
			return &metav1.Duration{Duration: d}, nil
		}
		return nil, nil
	}

	if config.Timeout, err =
		parseDuration(pipelineRunsConfigKeyTimeout); err != nil {
		return nil, err
	}

	if config.JenkinsfileRunnerPodSecurityContextRunAsUser, err =
		parseInt64(pipelineRunsConfigKeyPSCRunAsUser); err != nil {
		return nil, err
	}
	if config.JenkinsfileRunnerPodSecurityContextRunAsGroup, err =
		parseInt64(pipelineRunsConfigKeyPSCRunAsGroup); err != nil {
		return nil, err
	}

	if config.JenkinsfileRunnerPodSecurityContextFSGroup, err =
		parseInt64(pipelineRunsConfigKeyPSCFSGroup); err != nil {
		return nil, err
	}

	return config, nil
}
