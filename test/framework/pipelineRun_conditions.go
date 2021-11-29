package framework

import (
	"context"
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

// PipelineRunCheck is a Check for a PipelineRun
type PipelineRunCheck func(*api.PipelineRun) (bool, error)

// CreatePipelineRunCondition returns a WaitCondition for a PipelineRun with a dedicated PipelineCheck
func CreatePipelineRunCondition(pipelineRunToFind *api.PipelineRun, check PipelineRunCheck) WaitConditionFunc {
	return func(ctx context.Context) (bool, error) {
		fetcher := k8s.NewClientBasedPipelineRunFetcher(GetClientFactory(ctx).StewardV1alpha1())
		pipelineRun, err := fetcher.ByName(ctx, pipelineRunToFind.GetNamespace(), pipelineRunToFind.GetName())
		if err != nil {
			return true, err
		}
		if pipelineRun == nil {
			return true, fmt.Errorf("pipelinerun not found '%s/%s'", pipelineRunToFind.GetNamespace(), pipelineRunToFind.GetName())
		}
		return check(pipelineRun)
	}
}

// PipelineRunHasStateResult returns a PipelineRunCheck which Checks if a PipelineRun has a dedicated result
func PipelineRunHasStateResult(result api.Result) PipelineRunCheck {
	return func(pr *api.PipelineRun) (bool, error) {
		if pr.Status.Result == "" {
			return false, nil
		}
		if pr.Status.Result == result {
			return true, nil
		}
		return true, fmt.Errorf("unexpected result: expecting %q, got %q", result, pr.Status.Result)
	}
}

// PipelineRunMessageOnFinished returns a PipelineRunCheck which Checks if a PipelineRun has a dedicated message when it is in state finished
func PipelineRunMessageOnFinished(message string) PipelineRunCheck {
	return func(pr *api.PipelineRun) (bool, error) {
		if pr.Status.State == api.StateFinished {
			if pr.Status.Message == message {
				return true, nil
			}
			return true, fmt.Errorf("unexpected message: expecting %q, got %q", message, pr.Status.Message)

		}
		return false, nil
	}
}
