package runctl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward"
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
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

	// in general, the token of the above service account should not be automatically mounted into pods
	automountServiceAccountToken = false

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

type runInstanceTesting struct {
	cleanupStub                               func(context.Context) error
	copySecretsToRunNamespaceStub             func(context.Context) (string, []string, error)
	getSecretHelperStub                       func(string, corev1.SecretInterface) secrets.SecretHelper
	setupNetworkPolicyFromConfigStub          func(context.Context) error
	setupNetworkPolicyThatIsolatesAllPodsStub func(context.Context) error
	setupServiceAccountStub                   func(context.Context, string, []string) error
	setupStaticNetworkPoliciesStub            func(context.Context) error
	getServiceAccountSecretNameStub           func(context.Context) string
	createTektonTaskRunStub                   func(ctx context.Context) error
	prepareRunNamespaceStub                   func(ctx context.Context) error
}

type runInstance struct {
	pipelineRun        k8s.PipelineRun
	pipelineRunsConfig pipelineRunsConfigStruct
	runNamespace       string
	serviceAccount     *k8s.ServiceAccountWrap
}

// prepareRunNamespace creates a new namespace for the pipeline run
// and populates it with needed resources.
func (c *runInstance) prepareRunNamespace(ctx context.Context) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).prepareRunNamespaceStub != nil {
		return GetRunInstanceTesting(ctx).prepareRunNamespaceStub(ctx)
	}
	var err error

	c.runNamespace, err = k8s.GetNamespaceManager(ctx).Create("", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create run namespace")
	}

	c.pipelineRun.UpdateRunNamespace(c.runNamespace)

	// If something goes wrong while creating objects inside the namespaces, we delete everything.
	cleanupOnError := func() {
		if err != nil {
			c.cleanup(ctx)
		}
	}
	defer cleanupOnError()

	pipelineCloneSecretName, imagePullSecretNames, err := c.copySecretsToRunNamespace(ctx)
	if err != nil {
		return err
	}

	err = c.setupServiceAccount(ctx, pipelineCloneSecretName, imagePullSecretNames)
	if err != nil {
		return err
	}

	if err = c.setupStaticNetworkPolicies(ctx); err != nil {
		return err
	}

	return nil
}

func (c *runInstance) setupServiceAccount(ctx context.Context, pipelineCloneSecretName string, imagePullSecrets []string) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).setupServiceAccountStub != nil {
		return GetRunInstanceTesting(ctx).setupServiceAccountStub(ctx, pipelineCloneSecretName, imagePullSecrets)
	}
	// TODO: New Service Account Manager with Context
	factory := k8s.GetClientFactory(ctx)
	accountManager := k8s.NewServiceAccountManager(factory, c.runNamespace)
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
			serviceAccount.SetDoAutomountServiceAccountToken(automountServiceAccountToken)
			err = serviceAccount.Update()
			if err == nil {
				break // ...the retry loop
			}
			if k8serrors.IsConflict(err) {
				// resource version conflict -> retry update with latest version
				log.Printf(
					"retrying update of service account %q in namespace %q"+
						" after resource version conflict",
					serviceAccountName, c.runNamespace,
				)
			} else {
				return errors.Wrapf(err, "failed to update service account %q", serviceAccountName)
			}
		}
	}

	// grant role to service account
	_, err = serviceAccount.AddRoleBinding(runClusterRoleName, c.runNamespace)
	if err != nil {
		return errors.Wrapf(err,
			"failed to create role binding for service account %q in namespace %q",
			serviceAccountName, c.runNamespace,
		)
	}
	c.serviceAccount = serviceAccount
	return nil
}

func (c *runInstance) copySecretsToRunNamespace(ctx context.Context) (string, []string, error) {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).copySecretsToRunNamespaceStub != nil {
		return GetRunInstanceTesting(ctx).copySecretsToRunNamespaceStub(ctx)
	}

	targetClient := k8s.GetClientFactory(ctx).CoreV1().Secrets(c.runNamespace)
	secretHelper := c.getSecretHelper(ctx, targetClient)

	imagePullSecretNames, err := c.copyImagePullSecretsToRunNamespace(ctx, secretHelper)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy image pull secrets")
	}

	pipelineCloneSecretName, err := c.copyPipelineCloneSecretToRunNamespace(ctx, secretHelper)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline clone secret")
	}

	_, err = c.copyPipelineSecretsToRunNamespace(ctx, secretHelper)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to copy pipeline secrets")
	}

	return pipelineCloneSecretName, imagePullSecretNames, nil
}

func (c *runInstance) copyImagePullSecretsToRunNamespace(ctx context.Context, secretHelper secrets.SecretHelper) ([]string, error) {
	secretNames := c.pipelineRun.GetSpec().ImagePullSecrets
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer("tekton.dev/"),
		secrets.StripAnnotationsTransformer("jenkins.io/"),
		secrets.StripLabelsTransformer("jenkins.io/"),
		secrets.UniqueNameTransformer(),
	}
	return c.copySecrets(ctx, secretHelper, secretNames, secrets.DockerOnly, transformers...)
}

func (c *runInstance) copyPipelineCloneSecretToRunNamespace(ctx context.Context, secretHelper secrets.SecretHelper) (string, error) {
	secretName := c.pipelineRun.GetSpec().JenkinsFile.RepoAuthSecret
	if secretName == "" {
		return "", nil
	}
	repoServerURL, err := c.pipelineRun.GetPipelineRepoServerURL()
	if err != nil {
		// TODO: this method should not modify the pipeline run -> must be handled elsewhere
		c.pipelineRun.UpdateMessage(err.Error())
		c.pipelineRun.UpdateResult(v1alpha1.ResultErrorContent)
		return "", err
	}
	transformers := []secrets.SecretTransformer{
		secrets.StripAnnotationsTransformer("jenkins.io/"),
		secrets.StripLabelsTransformer("jenkins.io/"),
		secrets.UniqueNameTransformer(),
		secrets.SetAnnotationTransformer("tekton.dev/git-0", repoServerURL),
	}
	names, err := c.copySecrets(ctx, secretHelper, []string{secretName}, nil, transformers...)
	if err != nil {
		return "", err
	}
	return names[0], nil
}

func (c *runInstance) copyPipelineSecretsToRunNamespace(ctx context.Context, secretHelper secrets.SecretHelper) ([]string, error) {
	secretNames := c.pipelineRun.GetSpec().Secrets
	stripTektonAnnotationsTransformer := secrets.StripAnnotationsTransformer("tekton.dev/")
	return c.copySecrets(ctx, secretHelper, secretNames, nil, stripTektonAnnotationsTransformer)
}

func (c *runInstance) getSecretHelper(ctx context.Context, targetClient corev1.SecretInterface) secrets.SecretHelper {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).getSecretHelperStub != nil {
		return GetRunInstanceTesting(ctx).getSecretHelperStub(c.runNamespace, targetClient)
	}
	//TODO: Secret Helper creation with context
	secretProvider := secrets.GetSecretProvider(ctx)
	return secrets.NewSecretHelper(secretProvider, c.runNamespace, targetClient)
}

func (c *runInstance) copySecrets(ctx context.Context, secretHelper secrets.SecretHelper, secretNames []string, filter secrets.SecretFilter, transformers ...secrets.SecretTransformer) ([]string, error) {
	storedSecretNames, err := secretHelper.CopySecrets(secretNames, filter, transformers...)
	if err != nil {
		log.Printf("Cannot copy secrets %s for [%s]. Error: %s", secretNames, c.pipelineRun.String(), err)
		c.pipelineRun.UpdateMessage(err.Error())
		if secretHelper.IsNotFound(err) {
			c.pipelineRun.UpdateResult(v1alpha1.ResultErrorContent)
		} else {
			c.pipelineRun.UpdateResult(v1alpha1.ResultErrorInfra)
		}
		return storedSecretNames, err
	}
	return storedSecretNames, nil
}

func (c *runInstance) setupStaticNetworkPolicies(ctx context.Context) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).setupStaticNetworkPoliciesStub != nil {
		return GetRunInstanceTesting(ctx).setupStaticNetworkPoliciesStub(ctx)
	}

	if err := c.setupNetworkPolicyThatIsolatesAllPods(ctx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the network policy isolating all pods in namespace %q",
			c.runNamespace,
		)
	}
	if err := c.setupNetworkPolicyFromConfig(ctx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured network policy in namespace %q",
			c.runNamespace,
		)
	}
	return nil
}

func (c *runInstance) setupNetworkPolicyThatIsolatesAllPods(ctx context.Context) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).setupNetworkPolicyThatIsolatesAllPodsStub != nil {
		return GetRunInstanceTesting(ctx).setupNetworkPolicyThatIsolatesAllPodsStub(ctx)
	}

	policy := &networkingv1api.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: steward.GroupName + "--isolate-all-",
			Namespace:    c.runNamespace,
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

	policyIfce := k8s.GetClientFactory(ctx).NetworkingV1().NetworkPolicies(c.runNamespace)
	if _, err := policyIfce.Create(policy); err != nil {
		return errors.Wrap(err, "error when creating network policy")
	}

	return nil
}

func (c *runInstance) setupNetworkPolicyFromConfig(ctx context.Context) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).setupNetworkPolicyFromConfigStub != nil {
		return GetRunInstanceTesting(ctx).setupNetworkPolicyFromConfigStub(ctx)
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
		obj.SetNamespace(c.runNamespace)
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
		dynamicIfce := k8s.GetClientFactory(ctx).Dynamic().Resource(gvr).Namespace(c.runNamespace)
		if _, err := dynamicIfce.Create(obj, metav1.CreateOptions{}); err != nil {
			return errors.Wrap(err, "failed to create configured network policy")
		}
	}

	return nil
}

func (c *runInstance) volumesWithServiceAccountSecret(ctx context.Context) []corev1api.Volume {
	var mode int32 = 0644
	return []corev1api.Volume{
		corev1api.Volume{
			Name: "service-account-token",
			VolumeSource: corev1api.VolumeSource{
				Secret: &corev1api.SecretVolumeSource{
					SecretName:  c.getServiceAccountSecretName(ctx),
					DefaultMode: &mode,
				},
			},
		},
	}
}

func (c *runInstance) getServiceAccountSecretName(ctx context.Context) string {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).getServiceAccountSecretNameStub != nil {
		return GetRunInstanceTesting(ctx).getServiceAccountSecretNameStub(ctx)
	}
	k8s.EnsureServiceAccountTokenSecretRetriever(ctx)
	ret := k8s.GetServiceAccountTokenSecretRetriever(ctx)
	secret, err := ret.ForObj(ctx, c.serviceAccount.GetServiceAccount())
	if err != nil {
		return secret.GetName()
	} else {
		return ""
	}
}

func (c *runInstance) createTektonTaskRun(ctx context.Context) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).createTektonTaskRunStub != nil {
		return GetRunInstanceTesting(ctx).createTektonTaskRunStub(ctx)
	}
	var err error
	copyInt64Ptr := func(ptr *int64) *int64 {
		if ptr != nil {
			v := *ptr
			return &v
		}
		return nil
	}

	namespace := c.runNamespace

	tektonTaskRun := tekton.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonTaskRunName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotationPipelineRunKey: c.pipelineRun.GetKey(),
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
			PodTemplate: tekton.PodTemplate{
				SecurityContext: &corev1api.PodSecurityContext{
					RunAsUser:  copyInt64Ptr(c.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsUser),
					RunAsGroup: copyInt64Ptr(c.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup),
					FSGroup:    copyInt64Ptr(c.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextFSGroup),
				},
				Volumes: c.volumesWithServiceAccountSecret(ctx),
			},
		},
	}

	c.addTektonTaskRunParamsForPipeline(ctx, &tektonTaskRun)
	c.addTektonTaskRunParamsForLoggingElasticsearch(ctx, &tektonTaskRun)
	c.addTektonTaskRunParamsForRunDetails(ctx, &tektonTaskRun)
	tektonClient := k8s.GetClientFactory(ctx).TektonV1alpha1()
	_, err = tektonClient.TaskRuns(tektonTaskRun.GetNamespace()).Create(&tektonTaskRun)
	return err
}

func (c *runInstance) addTektonTaskRunParamsForRunDetails(
	ctx context.Context,
	tektonTaskRun *tekton.TaskRun,
) {
	spec := c.pipelineRun.GetSpec()
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

func (c *runInstance) addTektonTaskRunParamsForPipeline(
	ctx context.Context,
	tektonTaskRun *tekton.TaskRun,
) error {
	var err error

	spec := c.pipelineRun.GetSpec()
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

func (c *runInstance) addTektonTaskRunParamsForLoggingElasticsearch(
	ctx context.Context,
	tektonTaskRun *tekton.TaskRun,
) error {
	spec := c.pipelineRun.GetSpec()
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

func (c *runInstance) cleanup(ctx context.Context) error {
	if GetRunInstanceTesting(ctx) != nil && GetRunInstanceTesting(ctx).cleanupStub != nil {
		return GetRunInstanceTesting(ctx).cleanupStub(ctx)
	}

	namespace := c.runNamespace
	if namespace == "" {
		//TODO: Don't store on resource as message. Add it as event.
		c.pipelineRun.StoreErrorAsMessage(fmt.Errorf("Nothing to clean up as namespace not set"), "")
	} else {
		err := k8s.GetNamespaceManager(ctx).Delete(namespace)
		if err != nil {
			c.pipelineRun.StoreErrorAsMessage(err, "error deleting namespace")
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
