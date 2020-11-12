package builder

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRunOp is an operation which modifies a PipelineRun.
type PipelineRunOp func(*api.PipelineRun)

// PipelineRunSpecOp is an operation returning a modified PipelineSpec
type PipelineRunSpecOp func(api.PipelineSpec) api.PipelineSpec

// JenkinsFileOp is an operation returning a modified JenkinsFile
type JenkinsFileOp func(api.JenkinsFile) api.JenkinsFile

// PipelineRun creates a PipelineRun
// Any number of PipelineRunOps can be passed
func PipelineRun(prefix, namespace string, ops ...PipelineRunOp) *api.PipelineRun {
	run := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: prefix,
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
func JenkinsFileSpec(url, path string, ops ...JenkinsFileOp) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		spec.JenkinsFile = api.JenkinsFile{
			URL:      url,
			Revision: "master",
			Path:     path,
		}
		for _, op := range ops {
			spec.JenkinsFile = op(spec.JenkinsFile)
		}
		return spec
	}
}

// Revision creates a JeninsFileOp setting the revision of the jenkins file
func Revision(r string) JenkinsFileOp {
	return func(spec api.JenkinsFile) api.JenkinsFile {
		spec.Revision = r
		return spec
	}
}

// RepoAuthSecret creates a JenkinsFileOp setting the repo auth secret
func RepoAuthSecret(name string) JenkinsFileOp {
	return func(spec api.JenkinsFile) api.JenkinsFile {
		spec.RepoAuthSecret = name
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

// Secret creates a PipelineRunSpecOp which adds a Secret
func Secret(name string) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		secrets := spec.Secrets
		if secrets == nil {
			secrets = []string{name}
		} else {
			secrets = append(secrets, name)
		}
		spec.Secrets = secrets
		return spec
	}
}

// ImagePullSecret creates a PipelineRUnSpecOp which adds an Image Pull Secret
func ImagePullSecret(name string) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		secrets := spec.ImagePullSecrets
		if secrets == nil {
			secrets = []string{name}
		} else {
			secrets = append(secrets, name)
		}
		spec.ImagePullSecrets = secrets
		return spec
	}
}

// RunDetails creates a PipelineRunSpecOp which adds RunDetails
func RunDetails(jobName, cause string, sequenceNumber int32) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		spec.RunDetails = &api.PipelineRunDetails{
			JobName:        jobName,
			SequenceNumber: sequenceNumber,
			Cause:          cause,
		}
		return spec
	}
}

// Abort creates a PipelineRunSpecOp which adds Intent abort to the PipelineRun
func Abort() PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		spec.Intent = api.IntentAbort
		return spec
	}
}

// LoggingWithRunID creates a PipelineRunSpecOp which adds Logging to the PipelineRun
func LoggingWithRunID(runID *api.CustomJSON) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		logging := &api.Logging{
			Elasticsearch: &api.Elasticsearch{},
		}
		if spec.Logging != nil {
			logging = spec.Logging
		}

		if logging.Elasticsearch == nil {
			logging.Elasticsearch = &api.Elasticsearch{}
		}
		logging.Elasticsearch.RunID = runID
		spec.Logging = logging

		return spec
	}
}

// LoggingWithIndexURL creates a PipelineRunSpecOp which adds Logging to the PipelineRun with specific indexURL
func LoggingWithIndexURL(indexURL string) PipelineRunSpecOp {
	return func(spec api.PipelineSpec) api.PipelineSpec {
		logging := &api.Logging{
			Elasticsearch: &api.Elasticsearch{},
		}
		if spec.Logging != nil {
			logging = spec.Logging
		}

		if logging.Elasticsearch == nil {
			logging.Elasticsearch = &api.Elasticsearch{}
		}

		logging.Elasticsearch.IndexURL = indexURL
		spec.Logging = logging

		return spec
	}
}
