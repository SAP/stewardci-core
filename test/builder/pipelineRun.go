package builder

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRunOp is an operation which modifies a PipelineRun.
type PipelineRunOp func(*api.PipelineRun)

// PipelineRunSpecOp is an operation returning a modified PipelineSpec
type PipelineRunSpecOp func(api.PipelineSpec) api.PipelineSpec

// PipelineRun creates a PipelineRun
// Any number of PipelineRunOps can be passed
func PipelineRun(namespace string, ops ...PipelineRunOp) *api.PipelineRun {
	run := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "run-",
		},
	}
	for _, op := range ops {
		op(run)
	}
	return run
}

// PipelineRunSpec creates a PipelineRunSpec
// Any number of PipelineRunSpecOps can be passed
func PipelineRunSpec(ops ...PipelineRunSpecOp) PipelineRunOp {
	return func(run *api.PipelineRun) {
		spec := run.Spec
		for _, op := range ops {
			spec = op(spec)
		}
		run.Spec = spec

	}
}

// JenkinsFileSpec creates a JenkinsFileSpec
// TODO: introduce JenkinsFileSpecOp
func JenkinsFileSpec(url, revision, path string) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		spec.JenkinsFile = api.JenkinsFile{
			URL:      url,
			Revision: revision,
			Path:     path,
		}
		return spec

	}
}

// ArgSpec creates a PipelineRunSpecOp which adds an ArgSpec
func ArgSpec(key, value string) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		args := spec.Args
		if args == nil {
			args = map[string]string{key: value}
		} else {
			args[key] = value
		}
		spec.Args = args
		return spec
	}
}
