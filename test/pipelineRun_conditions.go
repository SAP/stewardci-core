package test

import (
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

type PipelineRunCheck func(k8s.PipelineRun) bool

func CreatePipelineRunCondition(pipelineRun *api.PipelineRun, check PipelineRunCheck, desc string) WaitCondition {
	return NewWaitCondition(func(clientFactory k8s.ClientFactory) (bool, error) {
		fetcher := k8s.NewPipelineRunFetcher(clientFactory)
		pipelineRun, err := fetcher.ByName(pipelineRun.GetNamespace(), pipelineRun.GetName())
		if err != nil {
			return true, err
		}
		result := check(pipelineRun)
		return result, nil
	},
		fmt.Sprintf("PRC_%s_%s_%s", pipelineRun.GetNamespace(), pipelineRun.GetName(), desc))
}

func PipelineRunHasStateResult(result api.Result) PipelineRunCheck {
	return func(pr k8s.PipelineRun) bool {
		return pr.GetStatus().Result == result
	}
}
