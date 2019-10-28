package runctl

import (
	"encoding/json"
	"fmt"

	"time"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	runNamespacePrefix       = "steward-run"
	runNamespaceRandomLength = 16
	serviceAccountName       = "run-bot"

	annotationPipelineRunKey = "steward.sap.com/pipeline-run-key"

	// tektonClusterTaskName is the name of the Tekton ClusterTask
	// that should be used to execute the Jenkinsfile Runner
	tektonClusterTaskName = "steward-jenkinsfile-runner"

	// tektonClusterTaskJenkinsfileRunnerStep is the name of the step
	// in the Tekton TaskRun that executes the Jenkinsfile Runner
	tektonClusterTaskJenkinsfileRunnerStep = "jenkinsfile-runner"

	// tektonTaskRun is the name of the Tekton TaskRun in each
	// run namespace.
	tektonTaskRunName = "steward-jenkinsfile-runner"
)

// RunManager manages runs
type RunManager interface {
	Start(pipelineRun k8s.PipelineRun) error
	GetRun(pipelineRun k8s.PipelineRun) (Run, error)
	Cleanup(pipelineRun k8s.PipelineRun) error
}

type runManager struct {
	secretProvider   k8s.SecretProvider
	factory          k8s.ClientFactory
	namespaceManager k8s.NamespaceManager
}

// NewRunManager creates a new RunManager.
func NewRunManager(factory k8s.ClientFactory, secretProvider k8s.SecretProvider, namespaceManager k8s.NamespaceManager) RunManager {
	return &runManager{
		secretProvider:   secretProvider,
		factory:          factory,
		namespaceManager: namespaceManager,
	}
}

// Start prepares the isolated environment for a new run and starts
// the run in this environment.
func (c *runManager) Start(pipelineRun k8s.PipelineRun) error {
	var err error

	err = c.prepareRunNamespace(pipelineRun)
	if err != nil {
		return err
	}
	err = c.createTektonTaskRun(pipelineRun)
	if err != nil {
		return err
	}

	return nil
}

// prepareRunNamespace creates a new namespace for the pipeline run
// and populates it with needed resource.
func (c *runManager) prepareRunNamespace(pipelineRun k8s.PipelineRun) error {
	var err error

	//Create Run Namespace
	runNamespace, err := c.namespaceManager.Create("", nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create run namespace.")
	}

	//Assign namespace to Run
	pipelineRun.UpdateRunNamespace(runNamespace)

	// If something goes wrong while creating objects inside the namespaces, we delete everything.
	cleanupOnError := func() {
		if err != nil {
			c.Cleanup(pipelineRun)
		}
	}
	defer cleanupOnError()

	//Copy secrets to Run Namespace
	pipelineCloneSecretName, err := c.copyPipelinePullSecret(pipelineRun)
	if err != nil {
		return errors.Wrap(err, "failed to copy pipeline pull secret")
	}

	secretNames := pipelineRun.GetSpec().Secrets
	stripTektonAnnotationsFunc := k8s.StripAnnotationsFunc("tekton")
	secretNames, err = c.copySecrets(runNamespace, secretNames, pipelineRun, nil, stripTektonAnnotationsFunc)
	if err != nil {
		return errors.Wrap(err, "failed to copy secrets")
	}

	imagePullSecrets := pipelineRun.GetSpec().ImagePullSecrets
	random, err := utils.RandomAlphaNumString(6)
	if err != nil {
		return err
	}
	transformers := []k8s.SecretTransformerType{
		stripTektonAnnotationsFunc,
		k8s.StripAnnotationsFunc("jenkins"),
		k8s.StripLabelsFunc("jenkins"),
		k8s.AppendNameSuffixFunc(random),
	}

	imagePullSecrets, err = c.copySecrets(runNamespace, imagePullSecrets, pipelineRun, k8s.DockerOnly, transformers...)

	//Create Service Account in Run Namespace
	accountManager := k8s.NewServiceAccountManager(c.factory, runNamespace)

	serviceAccount, err := accountManager.CreateServiceAccount(serviceAccountName, pipelineCloneSecretName, imagePullSecrets)
	if err != nil {
		return errors.Wrap(err, "failed to create service account.")
	}

	//Add Role Binding to Service Account
	_, err = serviceAccount.AddRoleBinding(runClusterRoleName, runNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to create role binding")
	}

	return nil
}

func (c *runManager) copyPipelinePullSecret(pipelineRun k8s.PipelineRun) (string, error) {
	pipelineCloneSecret := pipelineRun.GetSpec().JenkinsFile.Secret
	if pipelineCloneSecret == "" {
		return "", nil
	}
	random, err := utils.RandomAlphaNumString(6)
	if err != nil {
		return "", err
	}
	repoServer, err := pipelineRun.GetRepoServerURL()
	if err != nil {
		return "", err
	}
	transformers := []k8s.SecretTransformerType{
		k8s.StripAnnotationsFunc("jenkins"),
		k8s.StripLabelsFunc("jenkins"),
		k8s.AppendNameSuffixFunc(random),
		k8s.SetAnnotationFunc("tekton.dev/git-0", repoServer),
	}
	names, err := c.copySecrets(pipelineRun.GetRunNamespace(), []string{pipelineCloneSecret}, pipelineRun, nil, transformers...)
	if err != nil {
		return "", err
	}
	return names[0], nil
}

func (c *runManager) copySecrets(targetNamespace string, secretNames []string, pipelineRun k8s.PipelineRun, filter k8s.SecretFilterType, transformers ...k8s.SecretTransformerType) ([]string, error) {
	targetClient := c.factory.CoreV1().Secrets(targetNamespace)
	secretHelper := k8s.NewSecretHelper(c.secretProvider, targetNamespace, targetClient)

	storedSecretNames, err := secretHelper.CopySecrets(secretNames, filter, transformers...)
	if err != nil {
		pipelineRun.UpdateResult(v1alpha1.ResultErrorContent)
		pipelineRun.UpdateMessage(err.Error())
		return storedSecretNames, err
	}
	return storedSecretNames, nil
}

func (c *runManager) createTektonTaskRun(pipelineRun k8s.PipelineRun) error {
	var err error

	namespace := pipelineRun.GetRunNamespace()

	tektonTaskRun := tekton.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonTaskRunName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotationPipelineRunKey: pipelineRun.GetKey(),
			},
		},
		Spec: tekton.TaskRunSpec{
			ServiceAccount: serviceAccountName,
			TaskRef: &tekton.TaskRef{
				Kind: tekton.ClusterTaskKind,
				Name: tektonClusterTaskName,
			},
			Inputs: tekton.TaskRunInputs{
				Params: []tekton.Param{
					tektonStringParam("RUN_NAMESPACE", namespace),
				},
			},
			Timeout: toDuration(defaultBuildTimeout),
		},
	}

	c.addTektonTaskRunParamsForPipeline(pipelineRun, &tektonTaskRun)
	c.addTektonTaskRunParamsForLoggingElasticsearch(pipelineRun, &tektonTaskRun)

	tektonClient := c.factory.TektonV1alpha1()
	_, err = tektonClient.TaskRuns(tektonTaskRun.GetNamespace()).Create(&tektonTaskRun)
	return err
}

func (c *runManager) addTektonTaskRunParamsForPipeline(
	pipelineRun k8s.PipelineRun,
	tektonTaskRun *tekton.TaskRun,
) error {
	var err error

	spec := pipelineRun.GetSpec()
	pipeline := spec.JenkinsFile
	pipelineArgs := spec.Args
	pipelineArgsJSON := "{}"
	if pipelineArgs != nil {
		if pipelineArgsJSON, err = toJSONString(&pipelineArgs); err != nil {
			return err
		}
	}

	params := []tekton.Param{
		tektonStringParam("PIPELINE_GIT_URL", pipeline.URL),
		tektonStringParam("PIPELINE_GIT_REVISION", pipeline.Revision),
		tektonStringParam("PIPELINE_FILE", pipeline.Path),
		tektonStringParam("PIPELINE_PARAMS_JSON", pipelineArgsJSON),
	}

	tektonTaskRun.Spec.Inputs.Params = append(tektonTaskRun.Spec.Inputs.Params, params...)
	return nil
}

func (c *runManager) addTektonTaskRunParamsForLoggingElasticsearch(
	pipelineRun k8s.PipelineRun,
	tektonTaskRun *tekton.TaskRun,
) error {
	spec := pipelineRun.GetSpec()
	var params []tekton.Param

	if spec.Logging == nil || spec.Logging.Elasticsearch == nil {
		params = []tekton.Param{
			// overide the index URL hardcoded in the template by
			// the empty string to effective disable logging to
			// Elasticsearch
			tektonStringParam("PIPELINE_LOG_ELASTICSEARCH_INDEX_URL", ""),
		}
	} else {
		runIDJSON, err := toJSONString(&spec.Logging.Elasticsearch.RunID)
		if err != nil {
			return errors.WithMessage(err,
				"could not serialize spec.logging.elasticsearch.runid to JSON",
			)
		}

		params = []tekton.Param{
			tektonStringParam("PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON", runIDJSON),
			// use default values from build template for all other params
		}
	}

	tektonTaskRun.Spec.Inputs.Params = append(tektonTaskRun.Spec.Inputs.Params, params...)
	return nil
}

// GetRun based on a pipelineRun
func (c *runManager) GetRun(pipelineRun k8s.PipelineRun) (Run, error) {
	namespace := pipelineRun.GetRunNamespace()
	run, err := c.factory.TektonV1alpha1().TaskRuns(namespace).Get(tektonTaskRunName, metav1.GetOptions{})
	return NewRun(run), err
}

// Cleanup a run based on a pipelineRun
func (c *runManager) Cleanup(pipelineRun k8s.PipelineRun) error {
	namespace := pipelineRun.GetRunNamespace()
	if namespace == "" {
		pipelineRun.StoreErrorAsMessage(fmt.Errorf("Nothing to clean up as namespace not set"), "")
	} else {
		err := c.namespaceManager.Delete(namespace)
		if err != nil {
			pipelineRun.StoreErrorAsMessage(err, "error deleting namespace")
			return err
		}
	}
	pipelineRun.FinishState()
	return nil
}

func toJSONString(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", errors.Wrapf(err, "error while serializing to JSON: %v", err)
	}
	return string(bytes), nil
}

// toDuration converts a duration string (see time.ParseDuration) into
// a "k8s.io/apimachinery/pkg/apis/meta/v1".Duration object.
// Panics in case of errors.
func toDuration(duration string) *metav1.Duration {
	d, err := time.ParseDuration(duration)
	if err != nil {
		panic(err)
	}
	return &metav1.Duration{Duration: d}
}

func tektonStringParam(name string, value string) tekton.Param {
	return tekton.Param{
		Name: name,
		Value: tekton.ArrayOrString{
			Type:      tekton.ParamTypeString,
			StringVal: value,
		},
	}
}
