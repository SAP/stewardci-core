package framework

import (
	"context"
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

// PipelineRunCheck is a Check for a PipelineRun
type PipelineRunCheck func(k8s.PipelineRun) (bool, error)

// CreatePipelineRunCondition returns a WaitCondition for a PipelineRun with a dedicated PipelineCheck
func CreatePipelineRunCondition(PipelineRun *api.PipelineRun, Check PipelineRunCheck) WaitConditionFunc {
	return func(ctx context.Context) (bool, error) {
		fetcher := k8s.NewPipelineRunFetcher(GetClientFactory(ctx))
		PipelineRun, err := fetcher.ByName(PipelineRun.GetNamespace(), PipelineRun.GetName())
		if err != nil {
			return true, err
		}
		return Check(PipelineRun)
	}
}

// PipelineRunHasStateResult returns a PipelineRunCheck which Checks if a PipelineRun has a dedicated result
func PipelineRunHasStateResult(result api.Result) PipelineRunCheck {
	return func(pr k8s.PipelineRun) (bool, error) {
		if pr.GetStatus().Result == "" {
			return false, nil
		}
		if pr.GetStatus().Result == result {
			return true, nil
		}
		return true, fmt.Errorf("UnExpected result: Expecting %q, got %q", result, pr.GetStatus().Result)
	}
}

// PipelineRunMessageOnFinished returns a PipelineRunCheck which Checks if a PipelineRun has a dedicated message when it is in state finished
func PipelineRunMessageOnFinished(message string) PipelineRunCheck {
	return func(pr k8s.PipelineRun) (bool, error) {
		if pr.GetStatus().State == api.StateFinished {
			if pr.GetStatus().Message == message {
				return true, nil
			}
			return true, fmt.Errorf("UnExpected message: Expecting %q, got %q", message, pr.GetStatus().Message)

		}
		return false, nil
	}
}
