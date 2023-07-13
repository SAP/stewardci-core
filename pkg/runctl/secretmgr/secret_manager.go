package secretmgr

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/k8s"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

const (
	annotationPrefixJenkins = "jenkins.io/"
	annotationPrefixTekton  = "tekton.dev/"
)

// SecretManager manages the serets in a run-namespace for the controller.
type SecretManager struct {
	secretHelper secrets.SecretHelper
}

// NewSecretManager creates secrets in the run namesapce
func NewSecretManager(secretHelper secrets.SecretHelper) SecretManager {
	return SecretManager{
		secretHelper: secretHelper,
	}
}

// CopyAll copies the required secrets of a pipeline run to the respective run namespace.
func (s SecretManager) CopyAll(ctx context.Context, pipelineRun k8s.PipelineRun) (string, []string, error) {
	imagePullSecretNames, err := s.copyImagePullSecretsToRunNamespace(ctx, pipelineRun)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy image pull secrets")
	}

	pipelineCloneSecretName, err := s.copyPipelineCloneSecretToRunNamespace(ctx, pipelineRun)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline clone secret")
	}

	_, err = s.copyPipelineSecretsToRunNamespace(ctx, pipelineRun)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline secrets")
	}

	return pipelineCloneSecretName, imagePullSecretNames, nil
}

func (s SecretManager) copyImagePullSecretsToRunNamespace(ctx context.Context, pipelineRun k8s.PipelineRun) ([]string, error) {
	secretNames := pipelineRun.GetSpec().ImagePullSecrets
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer(annotationPrefixTekton),
		secrets.StripAnnotationsTransformer(annotationPrefixJenkins),
		secrets.StripLabelsTransformer(annotationPrefixJenkins),
		secrets.UniqueNameTransformer(),
	}
	return s.copySecrets(ctx, pipelineRun, secretNames, secrets.DockerOnly, transformers...)
}

func (s SecretManager) copyPipelineCloneSecretToRunNamespace(ctx context.Context, pipelineRun k8s.PipelineRun) (string, error) {
	secretName := pipelineRun.GetSpec().JenkinsFile.RepoAuthSecret
	if secretName == "" {
		return "", nil
	}
	repoServerURL, err := pipelineRun.GetValidatedJenkinsfileRepoServerURL()
	if err != nil {
		return "", serrors.Classify(err, v1alpha1.ResultErrorContent)
	}
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer(annotationPrefixJenkins),
		secrets.StripLabelsTransformer(annotationPrefixJenkins),
		secrets.UniqueNameTransformer(),
		secrets.SetAnnotationTransformer("tekton.dev/git-0", repoServerURL),
	}
	names, err := s.copySecrets(ctx, pipelineRun, []string{secretName}, nil, transformers...)
	if err != nil {
		return "", err
	}
	return names[0], nil
}

func (s SecretManager) copyPipelineSecretsToRunNamespace(ctx context.Context, pipelineRun k8s.PipelineRun) ([]string, error) {
	secretNames := pipelineRun.GetSpec().Secrets
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer(annotationPrefixTekton),
		secrets.RenameByAnnotationTransformer(v1alpha1.AnnotationSecretRename),
	}
	return s.copySecrets(ctx, pipelineRun, secretNames, nil, transformers...)
}

func (s SecretManager) copySecrets(ctx context.Context, pipelineRun k8s.PipelineRun, secretNames []string, filter secrets.SecretFilter, transformers ...secrets.SecretTransformer) ([]string, error) {
	logger := klog.FromContext(ctx)
	storedSecretNames, err := s.secretHelper.CopySecrets(ctx, secretNames, filter, transformers...)
	if err != nil {
		logger.Error(err, "Cannot copy secrets", "secrets", secretNames)
		if s.secretHelper.IsNotFound(err) || k8serrors.IsInvalid(err) || k8serrors.IsAlreadyExists(err) {
			err = serrors.Classify(err, v1alpha1.ResultErrorContent)
		} else {
			err = serrors.Classify(err, v1alpha1.ResultErrorInfra)
		}
		return storedSecretNames, err
	}
	return storedSecretNames, nil
}
