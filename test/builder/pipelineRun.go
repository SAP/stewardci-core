package builder

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PipelineRunOp func(*api.PipelineRun)

type PipelineRunSpecOp func(api.PipelineSpec) api.PipelineSpec

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

func PipelineRunSpec(ops ...PipelineRunSpecOp) PipelineRunOp {
	return func(run *api.PipelineRun) {
		spec := run.Spec
		for _, op := range ops {
		spec = op(spec)
		}
		run.Spec = spec

	}
}

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
