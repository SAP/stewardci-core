package secretmgr

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	klog "k8s.io/klog/v2"
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
func (s SecretManager) CopyAll(pipelineRun k8s.PipelineRun) (string, []string, error) {
	imagePullSecretNames, err := s.copyImagePullSecretsToRunNamespace(pipelineRun)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy image pull secrets")
	}

	pipelineCloneSecretName, err := s.copyPipelineCloneSecretToRunNamespace(pipelineRun)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline clone secret")
	}

	_, err = s.copyPipelineSecretsToRunNamespace(pipelineRun)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline secrets")
	}

	return pipelineCloneSecretName, imagePullSecretNames, nil
}

func (s SecretManager) copyImagePullSecretsToRunNamespace(pipelineRun k8s.PipelineRun) ([]string, error) {
	secretNames := pipelineRun.GetSpec().ImagePullSecrets
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer("tekton.dev/"),
		secrets.StripAnnotationsTransformer("jenkins.io/"),
		secrets.StripLabelsTransformer("jenkins.io/"),
		secrets.UniqueNameTransformer(),
	}
	return s.copySecrets(pipelineRun, secretNames, secrets.DockerOnly, transformers...)
}

func (s SecretManager) copyPipelineCloneSecretToRunNamespace(pipelineRun k8s.PipelineRun) (string, error) {
	secretName := pipelineRun.GetSpec().JenkinsFile.RepoAuthSecret
	if secretName == "" {
		return "", nil
	}
	repoServerURL, err := pipelineRun.GetPipelineRepoServerURL()
	if err != nil {
		// TODO: this method should not modify the pipeline run -> must be handled elsewhere
		pipelineRun.UpdateMessage(err.Error())
		pipelineRun.UpdateResult(v1alpha1.ResultErrorContent)
		return "", err
	}
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer("jenkins.io/"),
		secrets.StripLabelsTransformer("jenkins.io/"),
		secrets.UniqueNameTransformer(),
		secrets.SetAnnotationTransformer("tekton.dev/git-0", repoServerURL),
	}
	names, err := s.copySecrets(pipelineRun, []string{secretName}, nil, transformers...)
	if err != nil {
		return "", err
	}
	return names[0], nil
}

func (s SecretManager) copyPipelineSecretsToRunNamespace(pipelineRun k8s.PipelineRun) ([]string, error) {
	secretNames := pipelineRun.GetSpec().Secrets
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer("tekton.dev/"),
		secrets.RenameByAnnotationTransformer(v1alpha1.AnnotationSecretRename),
	}
	return s.copySecrets(pipelineRun, secretNames, nil, transformers...)
}

func (s SecretManager) copySecrets(pipelineRun k8s.PipelineRun, secretNames []string, filter secrets.SecretFilter, transformers ...secrets.SecretTransformer) ([]string, error) {
	storedSecretNames, err := s.secretHelper.CopySecrets(secretNames, filter, transformers...)
	if err != nil {
		klog.Errorf("Cannot copy secrets %s for [%s]. Error: %s", secretNames, pipelineRun.String(), err)
		pipelineRun.UpdateMessage(err.Error())
		if s.secretHelper.IsNotFound(err) || k8serrors.IsInvalid(err) || k8serrors.IsAlreadyExists(err) {
			pipelineRun.UpdateResult(v1alpha1.ResultErrorContent)
		} else {
			pipelineRun.UpdateResult(v1alpha1.ResultErrorInfra)
		}
		return storedSecretNames, err
	}
	return storedSecretNames, nil
}
