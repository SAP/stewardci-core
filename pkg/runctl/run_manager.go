package runctl

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward"
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	runi "github.com/SAP/stewardci-core/pkg/run"
	"github.com/pkg/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1api "k8s.io/api/core/v1"
	networkingv1api "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlserial "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	runNamespacePrefix       = "steward-run"
	runNamespaceRandomLength = 16
	serviceAccountName       = "default"

	annotationPipelineRunKey = steward.GroupName + "/pipeline-run-key"

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

type runManager struct {
	factory            k8s.ClientFactory
	pipelineRunsConfig pipelineRunsConfigStruct
	namespaceManager   k8s.NamespaceManager
	secretProvider     secrets.SecretProvider

	testing *runManagerTesting
}

type runManagerTesting struct {
	cleanupStub                               func(k8s.PipelineRun) error
	copySecretsToRunNamespaceStub             func(k8s.PipelineRun, string) (string, []string, error)
	getSecretHelperStub                       func(string, corev1.SecretInterface) secrets.SecretHelper
	setupNetworkPolicyFromConfigStub          func(string) error
	setupNetworkPolicyThatIsolatesAllPodsStub func(string) error
	setupServiceAccountStub                   func(string, string, []string) error
	setupStaticNetworkPoliciesStub            func(string) error
}

// NewRunManager creates a new RunManager.
func NewRunManager(factory k8s.ClientFactory, pipelineRunsConfig *pipelineRunsConfigStruct, secretProvider secrets.SecretProvider, namespaceManager k8s.NamespaceManager) runi.Manager {
	return &runManager{
		factory:            factory,
		pipelineRunsConfig: *pipelineRunsConfig,
		namespaceManager:   namespaceManager,
		secretProvider:     secretProvider,
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
// and populates it with needed resources.
func (c *runManager) prepareRunNamespace(pipelineRun k8s.PipelineRun) error {
	var err error

	runNamespace, err := c.namespaceManager.Create("", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create run namespace")
	}

	pipelineRun.UpdateRunNamespace(runNamespace)

	// If something goes wrong while creating objects inside the namespaces, we delete everything.
	cleanupOnError := func() {
		if err != nil {
			c.Cleanup(pipelineRun)
		}
	}
	defer cleanupOnError()

	pipelineCloneSecretName, imagePullSecretNames, err := c.copySecretsToRunNamespace(pipelineRun, runNamespace)
	if err != nil {
		return err
	}

	if err = c.setupServiceAccount(runNamespace, pipelineCloneSecretName, imagePullSecretNames); err != nil {
		return err
	}

	if err = c.setupStaticNetworkPolicies(runNamespace); err != nil {
		return err
	}

	return nil
}

func (c *runManager) setupServiceAccount(runNamespace string, pipelineCloneSecretName string, imagePullSecrets []string) error {
	if c.testing != nil && c.testing.setupServiceAccountStub != nil {
		return c.testing.setupServiceAccountStub(runNamespace, pipelineCloneSecretName, imagePullSecrets)
	}

	accountManager := k8s.NewServiceAccountManager(c.factory, runNamespace)
	serviceAccount, err := accountManager.CreateServiceAccount(serviceAccountName, pipelineCloneSecretName, imagePullSecrets)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "failed to create service account %q", serviceAccountName)
		}

		// service account exists already, so we need to attach secrets to it
		for { // retry loop
			serviceAccount, err = accountManager.GetServiceAccount(serviceAccountName)
			if err != nil {
				return errors.Wrapf(err, "failed to get service account %q", serviceAccountName)
			}
			if pipelineCloneSecretName != "" {
				serviceAccount.AttachSecrets(pipelineCloneSecretName)
			}
			serviceAccount.AttachImagePullSecrets(imagePullSecrets...)
			err = serviceAccount.Update()
			if err == nil {
				break // ...the retry loop
			}
			if k8serrors.IsConflict(err) {
				// resource version conflict -> retry update with latest version
				log.Printf(
					"retrying update of service account %q in namespace %q"+
						" after resource version conflict",
					serviceAccountName, runNamespace,
				)
			} else {
				return errors.Wrapf(err, "failed to update service account %q", serviceAccountName)
			}
		}
	}

	// grant role to service account
	_, err = serviceAccount.AddRoleBinding(runClusterRoleName, runNamespace)
	if err != nil {
		return errors.Wrapf(err,
			"failed to create role binding for service account %q in namespace %q",
			serviceAccountName, runNamespace,
		)
	}

	return nil
}

func (c *runManager) copySecretsToRunNamespace(pipelineRun k8s.PipelineRun, runNamespace string) (string, []string, error) {
	if c.testing != nil && c.testing.copySecretsToRunNamespaceStub != nil {
		return c.testing.copySecretsToRunNamespaceStub(pipelineRun, runNamespace)
	}

	targetClient := c.factory.CoreV1().Secrets(runNamespace)
	secretHelper := c.getSecretHelper(runNamespace, targetClient)

	imagePullSecretNames, err := c.copyImagePullSecretsToRunNamespace(pipelineRun, secretHelper)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy image pull secrets")
	}

	pipelineCloneSecretName, err := c.copyPipelineCloneSecretToRunNamespace(pipelineRun, secretHelper)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline clone secret")
	}

	_, err = c.copyPipelineSecretsToRunNamespace(pipelineRun, secretHelper)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline secrets")
	}

	return pipelineCloneSecretName, imagePullSecretNames, nil
}

func (c *runManager) copyImagePullSecretsToRunNamespace(pipelineRun k8s.PipelineRun, secretHelper secrets.SecretHelper) ([]string, error) {
	secretNames := pipelineRun.GetSpec().ImagePullSecrets
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer("tekton.dev/"),
		secrets.StripAnnotationsTransformer("jenkins.io/"),
		secrets.StripLabelsTransformer("jenkins.io/"),
		secrets.UniqueNameTransformer(),
	}
	return c.copySecrets(secretHelper, secretNames, pipelineRun, secrets.DockerOnly, transformers...)
}

func (c *runManager) copyPipelineCloneSecretToRunNamespace(pipelineRun k8s.PipelineRun, secretHelper secrets.SecretHelper) (string, error) {
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
	names, err := c.copySecrets(secretHelper, []string{secretName}, pipelineRun, nil, transformers...)
	if err != nil {
		return "", err
	}
	return names[0], nil
}

func (c *runManager) copyPipelineSecretsToRunNamespace(pipelineRun k8s.PipelineRun, secretHelper secrets.SecretHelper) ([]string, error) {
	secretNames := pipelineRun.GetSpec().Secrets
	stripTektonAnnotationsTransformer := secrets.StripAnnotationsTransformer("tekton.dev/")
	return c.copySecrets(secretHelper, secretNames, pipelineRun, nil, stripTektonAnnotationsTransformer)
}

func (c *runManager) getSecretHelper(runNamespace string, targetClient corev1.SecretInterface) secrets.SecretHelper {
	if c.testing != nil && c.testing.getSecretHelperStub != nil {
		return c.testing.getSecretHelperStub(runNamespace, targetClient)
	}
	return secrets.NewSecretHelper(c.secretProvider, runNamespace, targetClient)
}

func (c *runManager) copySecrets(secretHelper secrets.SecretHelper, secretNames []string, pipelineRun k8s.PipelineRun, filter secrets.SecretFilter, transformers ...secrets.SecretTransformer) ([]string, error) {
	storedSecretNames, err := secretHelper.CopySecrets(secretNames, filter, transformers...)
	if err != nil {
		log.Printf("Cannot copy secrets %s for [%s]. Error: %s", secretNames, pipelineRun.String(), err)
		pipelineRun.UpdateMessage(err.Error())
		if secretHelper.IsNotFound(err) {
			pipelineRun.UpdateResult(v1alpha1.ResultErrorContent)
		} else {
			pipelineRun.UpdateResult(v1alpha1.ResultErrorInfra)
		}
		return storedSecretNames, err
	}
	return storedSecretNames, nil
}

func (c *runManager) setupStaticNetworkPolicies(runNamespace string) error {
	if c.testing != nil && c.testing.setupStaticNetworkPoliciesStub != nil {
		return c.testing.setupStaticNetworkPoliciesStub(runNamespace)
	}

	if err := c.setupNetworkPolicyThatIsolatesAllPods(runNamespace); err != nil {
		return errors.Wrapf(err,
			"failed to set up the network policy isolating all pods in namespace %q",
			runNamespace,
		)
	}
	if err := c.setupNetworkPolicyFromConfig(runNamespace); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured network policy in namespace %q",
			runNamespace,
		)
	}
	return nil
}

func (c *runManager) setupNetworkPolicyThatIsolatesAllPods(runNamespace string) error {
	if c.testing != nil && c.testing.setupNetworkPolicyThatIsolatesAllPodsStub != nil {
		return c.testing.setupNetworkPolicyThatIsolatesAllPodsStub(runNamespace)
	}

	policy := &networkingv1api.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: steward.GroupName + "--isolate-all-",
			Namespace:    runNamespace,
			Labels: map[string]string{
				v1alpha1.LabelSystemManaged: "",
			},
		},
		Spec: networkingv1api.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // select all pods from namespace
			PolicyTypes: []networkingv1api.PolicyType{
				networkingv1api.PolicyTypeEgress,
				networkingv1api.PolicyTypeIngress,
			},
		},
	}

	policyIfce := c.factory.NetworkingV1().NetworkPolicies(runNamespace)
	if _, err := policyIfce.Create(policy); err != nil {
		return errors.Wrap(err, "error when creating network policy")
	}

	return nil
}

func (c *runManager) setupNetworkPolicyFromConfig(runNamespace string) error {
	if c.testing != nil && c.testing.setupNetworkPolicyFromConfigStub != nil {
		return c.testing.setupNetworkPolicyFromConfigStub(runNamespace)
	}

	expectedGroupKind := schema.GroupKind{
		Group: networkingv1api.GroupName,
		Kind:  "NetworkPolicy",
	}

	policyStr := c.pipelineRunsConfig.NetworkPolicy
	if policyStr == "" {
		return nil
	}

	var obj *unstructured.Unstructured

	// decode
	{
		// We don't assume a specific resource version so that users can configure
		// whatever the K8s apiserver understands.
		yamlSerializer := yamlserial.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		o, err := runtime.Decode(yamlSerializer, []byte(policyStr))
		if err != nil {
			return errors.Wrap(err, "failed to decode configured network policy")
		}
		gvk := o.GetObjectKind().GroupVersionKind()
		if gvk.GroupKind() != expectedGroupKind {
			return errors.Errorf(
				"configured network policy does not denote a %q but a %q",
				expectedGroupKind.String(), gvk.GroupKind().String(),
			)
		}
		obj = o.(*unstructured.Unstructured)
	}

	// set metadata
	{
		// ignore any existing metadata to prevent side effects
		delete(obj.Object, "metadata")

		obj.SetGenerateName(steward.GroupName + "--configured-")
		obj.SetNamespace(runNamespace)
		obj.SetLabels(map[string]string{
			v1alpha1.LabelSystemManaged: "",
		})
	}

	// create resource object
	{
		gvr := schema.GroupVersionResource{
			Group:    expectedGroupKind.Group,
			Version:  obj.GetObjectKind().GroupVersionKind().Version,
			Resource: "networkpolicies",
		}
		dynamicIfce := c.factory.Dynamic().Resource(gvr).Namespace(runNamespace)
		if _, err := dynamicIfce.Create(obj, metav1.CreateOptions{}); err != nil {
			return errors.Wrap(err, "failed to create configured network policy")
		}
	}

	return nil
}

func (c *runManager) createTektonTaskRun(pipelineRun k8s.PipelineRun) error {
	var err error

	copyInt64Ptr := func(ptr *int64) *int64 {
		if ptr != nil {
			v := *ptr
			return &v
		}
		return nil
	}

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
			ServiceAccountName: serviceAccountName,
			TaskRef: &tekton.TaskRef{
				Kind: tekton.ClusterTaskKind,
				Name: tektonClusterTaskName,
			},
			Inputs: tekton.TaskRunInputs{
				Params: []tekton.Param{
					tektonStringParam("RUN_NAMESPACE", namespace),
				},
			},
			// use default timeout from tekton
			// Timeout: toDuration(defaultBuildTimeout),

			// Always set a non-empty pod template even if we don't have
			// values to set. Otherwise the Tekton default pod template
			// would be used only in such cases but not if we have values
			// to set.
			PodTemplate: &tekton.PodTemplate{
				SecurityContext: &corev1api.PodSecurityContext{
					RunAsUser:  copyInt64Ptr(c.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsUser),
					RunAsGroup: copyInt64Ptr(c.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup),
					FSGroup:    copyInt64Ptr(c.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextFSGroup),
				},
			},
		},
	}

	c.addTektonTaskRunParamsForPipeline(pipelineRun, &tektonTaskRun)
	c.addTektonTaskRunParamsForLoggingElasticsearch(pipelineRun, &tektonTaskRun)
	c.addTektonTaskRunParamsForRunDetails(pipelineRun, &tektonTaskRun)
	tektonClient := c.factory.TektonV1alpha1()
	_, err = tektonClient.TaskRuns(tektonTaskRun.GetNamespace()).Create(&tektonTaskRun)
	return err
}

func (c *runManager) addTektonTaskRunParamsForRunDetails(
	pipelineRun k8s.PipelineRun,
	tektonTaskRun *tekton.TaskRun,
) {
	spec := pipelineRun.GetSpec()
	details := spec.RunDetails
	if details != nil {
		params := []tekton.Param{}
		if details.JobName != "" {
			params = append(params, tektonStringParam("JOB_NAME", details.JobName))
		}
		if details.SequenceNumber > 0 {
			params = append(params, tektonStringParam("RUN_NUMBER", fmt.Sprintf("%d", details.SequenceNumber)))
		}
		if details.Cause != "" {
			params = append(params, tektonStringParam("RUN_CAUSE", details.Cause))
		}
		tektonTaskRun.Spec.Inputs.Params = append(tektonTaskRun.Spec.Inputs.Params, params...)
	}
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
func (c *runManager) GetRun(pipelineRun k8s.PipelineRun) (runi.Run, error) {
	namespace := pipelineRun.GetRunNamespace()
	run, err := c.factory.TektonV1alpha1().TaskRuns(namespace).Get(tektonTaskRunName, metav1.GetOptions{})
	return NewRun(run), err
}

// Cleanup a run based on a pipelineRun
func (c *runManager) Cleanup(pipelineRun k8s.PipelineRun) error {
	if c.testing != nil && c.testing.cleanupStub != nil {
		return c.testing.cleanupStub(pipelineRun)
	}

	namespace := pipelineRun.GetRunNamespace()
	if namespace == "" {
		//TODO: Don't store on resource as message. Add it as event.
		pipelineRun.StoreErrorAsMessage(fmt.Errorf("Nothing to clean up as namespace not set"), "")
	} else {
		err := c.namespaceManager.Delete(namespace)
		if err != nil {
			pipelineRun.StoreErrorAsMessage(err, "error deleting namespace")
			return err
		}
	}
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
