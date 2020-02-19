package runctl

import (
	"github.com/SAP/stewardci-core/pkg/k8s"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	pipelineRunsConfigMapName          = "steward-pipelineruns"
	pipelineRunsConfigKeyNetworkPolicy = "networkPolicy"
)

type pipelineRunsConfigStruct struct {
	NetworkPolicy string
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
	}

	return config, nil
}
