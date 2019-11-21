package test

import (
	"context"
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

// PipelineRunCheck is a check for a PipelineRun
type PipelineRunCheck func(k8s.PipelineRun) (bool, error)

// CreatePipelineRunCondition returns a WaitCondition for a pipelineRun with a dedicated PipelineCheck
func CreatePipelineRunCondition(pipelineRun *api.PipelineRun, check PipelineRunCheck) WaitConditionFunc {
	return func(ctx context.Context) (bool, error) {
		fetcher := k8s.NewPipelineRunFetcher(GetClientFactory(ctx))
		pipelineRun, err := fetcher.ByName(pipelineRun.GetNamespace(), pipelineRun.GetName())
		if err != nil {
			return true, err
		}
		return check(pipelineRun)
	}
}

// PipelineRunHasStateResult returns a PipelineRunCheck which checks if a pipelineRun has a dedicated result
func PipelineRunHasStateResult(result api.Result) PipelineRunCheck {
	return func(pr k8s.PipelineRun) (bool, error) {
		if pr.GetStatus().Result == "" {
			return false, nil
		}
		if pr.GetStatus().Result == result {
			return true, nil
		}
		return true, fmt.Errorf("Unexpected result: expecting %q, got %q", result, pr.GetStatus().Result)
	}
}

// PipelineRunMessageOnFinished returns a PipelineRunCheck which checks if a pipelineRun has a dedicated message when it is in state finished
func PipelineRunMessageOnFinished(message string) PipelineRunCheck {
	return func(pr k8s.PipelineRun) (bool, error) {
		if pr.GetStatus().State == api.StateFinished {
			if pr.GetStatus().Message == message {
				return true, nil
			}
			return true, fmt.Errorf("Unexpected message: expecting %q, got %q", message, pr.GetStatus().Message)

		}
		return false, nil
	}
}
