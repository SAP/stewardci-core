package runctl

import (
	"strconv"

	"github.com/SAP/stewardci-core/pkg/k8s"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	pipelineRunsConfigMapName          = "steward-pipelineruns"
	pipelineRunsConfigKeyNetworkPolicy = "networkPolicy"
	pipelineRunsConfigKeyLimitRange    = "limitRange"
	pipelineRunsConfigKeyResourceQuota = "resourceQuota"
	pipelineRunsConfigKeyPSCRunAsUser  = "jenkinsfileRunner.podSecurityContext.runAsUser"
	pipelineRunsConfigKeyPSCRunAsGroup = "jenkinsfileRunner.podSecurityContext.runAsGroup"
	pipelineRunsConfigKeyPSCFSGroup    = "jenkinsfileRunner.podSecurityContext.fsGroup"
)

type pipelineRunsConfigStruct struct {
	NetworkPolicy                                 string
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
		return &pipelineRunsConfigStruct{}, nil
	}
	if err != nil {
		return nil, err
	}

	config := &pipelineRunsConfigStruct{
		NetworkPolicy: configMap.Data[pipelineRunsConfigKeyNetworkPolicy],
		LimitRange:    configMap.Data[pipelineRunsConfigKeyLimitRange],
		ResourceQuota: configMap.Data[pipelineRunsConfigKeyResourceQuota],
	}

	parseInt64 := func(key string) (*int64, error) {
		if strVal, ok := configMap.Data[key]; ok && strVal != "" {
			intVal, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot parse configuration value %q", key)
			}
			return &intVal, nil
		}
		return nil, nil
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
