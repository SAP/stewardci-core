package runctl

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/featureflag"
	"github.com/SAP/stewardci-core/pkg/k8s"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	runifc "github.com/SAP/stewardci-core/pkg/runctl/run"
	"github.com/SAP/stewardci-core/pkg/runctl/secretmgr"
	slabels "github.com/SAP/stewardci-core/pkg/stewardlabels"
	"github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1api "k8s.io/api/core/v1"
	networkingv1api "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlserial "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/util/retry"
	klog "k8s.io/klog/v2"
)

const (
	runNamespacePrefix       = "steward-run"
	runNamespaceRandomLength = 5
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

type runManager struct {
	factory        k8s.ClientFactory
	secretProvider secrets.SecretProvider
	metrics        metrics.Metrics

	testing *runManagerTesting
}

type runManagerTesting struct {
	cleanupStub                               func(*runContext) error
	copySecretsToRunNamespaceStub             func(*runContext) (string, []string, error)
	createTektonTaskRunStub                   func(*runContext) error
	getSecretManagerStub                      func(*runContext) runifc.SecretManager
	getServiceAccountSecretNameStub           func(*runContext) string
	prepareRunNamespaceStub                   func(*runContext) error
	setupLimitRangeFromConfigStub             func(*runContext) error
	setupNetworkPolicyFromConfigStub          func(*runContext) error
	setupNetworkPolicyThatIsolatesAllPodsStub func(*runContext) error
	setupResourceQuotaFromConfigStub          func(*runContext) error
	setupServiceAccountStub                   func(*runContext, string, []string) error
	setupStaticLimitRangeStub                 func(*runContext) error
	setupStaticNetworkPoliciesStub            func(*runContext) error
	setupStaticResourceQuotaStub              func(*runContext) error
}

type runContext struct {
	pipelineRun        k8s.PipelineRun
	pipelineRunsConfig *cfg.PipelineRunsConfigStruct
	runNamespace       string
	auxNamespace       string
	serviceAccount     *k8s.ServiceAccountWrap
}

// newRunManager creates a new runManager.
func newRunManager(factory k8s.ClientFactory, secretProvider secrets.SecretProvider, metrics metrics.Metrics) *runManager {
	return &runManager{
		factory:        factory,
		secretProvider: secretProvider,
		metrics:        metrics,
	}
}

// Start prepares the isolated environment for a new run and starts
// the run in this environment.
func (c *runManager) Start(pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) (namespace string, auxNamespace string, err error) {

	ctx := &runContext{
		pipelineRun:        pipelineRun,
		pipelineRunsConfig: pipelineRunsConfig,
		runNamespace:       pipelineRun.GetRunNamespace(),
		auxNamespace:       pipelineRun.GetAuxNamespace(),
	}
	err = c.cleanupNamespaces(ctx)
	if err != nil {
		return "", "", err
	}

	// If something goes wrong while creating objects inside the namespaces, we delete everything.
	defer func() {
		if err != nil {
			c.cleanupNamespaces(ctx) // clean-up ignoring error
		}
	}()

	err = c.prepareRunNamespace(ctx)
	if err != nil {
		return "", "", err
	}

	return ctx.runNamespace, ctx.auxNamespace, c.createTektonTaskRun(ctx)
}

// prepareRunNamespace creates a new namespace for the pipeline run
// and populates it with needed resources.
func (c *runManager) prepareRunNamespace(ctx *runContext) error {

	if c.testing != nil && c.testing.prepareRunNamespaceStub != nil {
		return c.testing.prepareRunNamespaceStub(ctx)
	}

	var err error

	randName, err := utils.RandomAlphaNumString(runNamespaceRandomLength)
	if err != nil {
		return err
	}

	ctx.runNamespace, err = c.createNamespace(ctx, "main", randName)
	if err != nil {
		return err
	}

	if featureflag.CreateAuxNamespaceIfUnused.Enabled() {
		ctx.auxNamespace, err = c.createNamespace(ctx, "aux", randName)
		if err != nil {
			return err
		}
	}

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

	if err = c.setupStaticLimitRange(ctx); err != nil {
		return err
	}

	if err = c.setupStaticResourceQuota(ctx); err != nil {
		return err
	}

	return nil
}

func (c *runManager) setupServiceAccount(ctx *runContext, pipelineCloneSecretName string, imagePullSecrets []string) error {
	if c.testing != nil && c.testing.setupServiceAccountStub != nil {
		return c.testing.setupServiceAccountStub(ctx, pipelineCloneSecretName, imagePullSecrets)
	}

	accountManager := k8s.NewServiceAccountManager(c.factory, ctx.runNamespace)
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
				klog.V(4).Infof(
					"retrying update of service account %q in namespace %q"+
						" after resource version conflict",
					serviceAccountName, ctx.runNamespace,
				)
			} else {
				return errors.Wrapf(err, "failed to update service account %q", serviceAccountName)
			}
		}
	}

	// grant role to service account
	_, err = serviceAccount.AddRoleBinding(runClusterRoleName, ctx.runNamespace)
	if err != nil {
		return errors.Wrapf(err,
			"failed to create role binding for service account %q in namespace %q",
			serviceAccountName, ctx.runNamespace,
		)
	}
	ctx.serviceAccount = serviceAccount
	return nil
}

func (c *runManager) copySecretsToRunNamespace(ctx *runContext) (string, []string, error) {
	if c.testing != nil && c.testing.copySecretsToRunNamespaceStub != nil {
		return c.testing.copySecretsToRunNamespaceStub(ctx)
	}
	return c.getSecretManager(ctx).CopyAll(ctx.pipelineRun)
}

func (c *runManager) getSecretManager(ctx *runContext) runifc.SecretManager {
	if c.testing != nil && c.testing.getSecretManagerStub != nil {
		return c.testing.getSecretManagerStub(ctx)
	}
	targetClient := c.factory.CoreV1().Secrets(ctx.runNamespace)
	secretHelper := secrets.NewSecretHelper(c.secretProvider, ctx.runNamespace, targetClient)
	return secretmgr.NewSecretManager(secretHelper)
}

func (c *runManager) setupStaticNetworkPolicies(ctx *runContext) error {
	if c.testing != nil && c.testing.setupStaticNetworkPoliciesStub != nil {
		return c.testing.setupStaticNetworkPoliciesStub(ctx)
	}

	if err := c.setupNetworkPolicyThatIsolatesAllPods(ctx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the network policy isolating all pods in namespace %q",
			ctx.runNamespace,
		)
	}
	if err := c.setupNetworkPolicyFromConfig(ctx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured network policy in namespace %q",
			ctx.runNamespace,
		)
	}
	return nil
}

func (c *runManager) setupNetworkPolicyThatIsolatesAllPods(ctx *runContext) error {
	if c.testing != nil && c.testing.setupNetworkPolicyThatIsolatesAllPodsStub != nil {
		return c.testing.setupNetworkPolicyThatIsolatesAllPodsStub(ctx)
	}

	policy := &networkingv1api.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: steward.GroupName + "--isolate-all-",
			Namespace:    ctx.runNamespace,
		},
		Spec: networkingv1api.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // select all pods from namespace
			PolicyTypes: []networkingv1api.PolicyType{
				networkingv1api.PolicyTypeEgress,
				networkingv1api.PolicyTypeIngress,
			},
		},
	}

	slabels.LabelAsSystemManaged(policy)

	policyIfce := c.factory.NetworkingV1().NetworkPolicies(ctx.runNamespace)
	if _, err := policyIfce.Create(policy); err != nil {
		return errors.Wrap(err, "error when creating network policy")
	}

	return nil
}

func (c *runManager) setupNetworkPolicyFromConfig(ctx *runContext) error {
	if c.testing != nil && c.testing.setupNetworkPolicyFromConfigStub != nil {
		return c.testing.setupNetworkPolicyFromConfigStub(ctx)
	}

	networkProfile := ctx.pipelineRunsConfig.DefaultNetworkProfile

	spec := ctx.pipelineRun.GetSpec()
	if spec.Profiles != nil && spec.Profiles.Network != "" {
		networkProfile = spec.Profiles.Network

		if _, exists := ctx.pipelineRunsConfig.NetworkPolicies[networkProfile]; !exists {
			return serrors.Classify(fmt.Errorf("network profile %q does not exist", networkProfile), stewardv1alpha1.ResultErrorConfig)
		}
	}

	if networkProfile == "" {
		return nil
	}

	expectedGroupKind := schema.GroupKind{
		Group: networkingv1api.GroupName,
		Kind:  "NetworkPolicy",
	}
	manifestYAMLStr := ctx.pipelineRunsConfig.NetworkPolicies[networkProfile]

	return c.createResource(manifestYAMLStr, "networkpolicies", "network policy", expectedGroupKind, ctx)
}

func (c *runManager) setupStaticLimitRange(ctx *runContext) error {
	if c.testing != nil && c.testing.setupStaticLimitRangeStub != nil {
		return c.testing.setupStaticLimitRangeStub(ctx)
	}

	if err := c.setupLimitRangeFromConfig(ctx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured limit range in namespace %q",
			ctx.runNamespace,
		)
	}

	return nil
}

func (c *runManager) setupLimitRangeFromConfig(ctx *runContext) error {
	if c.testing != nil && c.testing.setupLimitRangeFromConfigStub != nil {
		return c.testing.setupLimitRangeFromConfigStub(ctx)
	}

	expectedGroupKind := schema.GroupKind{
		Group: "",
		Kind:  "LimitRange",
	}

	configStr := ctx.pipelineRunsConfig.LimitRange
	if configStr == "" {
		return nil
	}

	return c.createResource(configStr, "limitranges", "limit range", expectedGroupKind, ctx)
}

func (c *runManager) setupStaticResourceQuota(ctx *runContext) error {
	if c.testing != nil && c.testing.setupStaticResourceQuotaStub != nil {
		return c.testing.setupStaticResourceQuotaStub(ctx)
	}

	if err := c.setupResourceQuotaFromConfig(ctx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured resource quota in namespace %q",
			ctx.runNamespace,
		)
	}

	return nil
}

func (c *runManager) setupResourceQuotaFromConfig(ctx *runContext) error {
	if c.testing != nil && c.testing.setupResourceQuotaFromConfigStub != nil {
		return c.testing.setupResourceQuotaFromConfigStub(ctx)
	}

	expectedGroupKind := schema.GroupKind{
		Group: "",
		Kind:  "ResourceQuota",
	}

	configStr := ctx.pipelineRunsConfig.ResourceQuota
	if configStr == "" {
		return nil
	}

	return c.createResource(configStr, "resourcequotas", "resource quota", expectedGroupKind, ctx)
}

func (c *runManager) createResource(configStr string, resource string, resourceDisplayName string, expectedGroupKind schema.GroupKind, ctx *runContext) error {
	var obj *unstructured.Unstructured

	// decode
	{
		// We don't assume a specific resource version so that users can configure
		// whatever the K8s apiserver understands.
		yamlSerializer := yamlserial.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		o, err := runtime.Decode(yamlSerializer, []byte(configStr))
		if err != nil {
			return errors.Wrapf(err, "failed to decode configured %s", resourceDisplayName)
		}
		gvk := o.GetObjectKind().GroupVersionKind()
		if gvk.GroupKind() != expectedGroupKind {
			return errors.Errorf(
				"configured %s does not denote a %q but a %q",
				resourceDisplayName, expectedGroupKind.String(), gvk.GroupKind().String(),
			)
		}
		obj = o.(*unstructured.Unstructured)
	}

	// set metadata
	{
		// ignore any existing metadata to prevent side effects
		delete(obj.Object, "metadata")

		obj.SetGenerateName(steward.GroupName + "--configured-")
		obj.SetNamespace(ctx.runNamespace)

		slabels.LabelAsSystemManaged(obj)
	}

	// create resource object
	{
		gvr := schema.GroupVersionResource{
			Group:    expectedGroupKind.Group,
			Version:  obj.GetObjectKind().GroupVersionKind().Version,
			Resource: resource,
		}
		dynamicIfce := c.factory.Dynamic().Resource(gvr).Namespace(ctx.runNamespace)
		if _, err := dynamicIfce.Create(obj, metav1.CreateOptions{}); err != nil {
			return errors.Wrapf(err, "failed to create configured %s", resourceDisplayName)
		}
	}

	return nil
}

func (c *runManager) volumesWithServiceAccountSecret(ctx *runContext) []corev1api.Volume {
	var mode int32 = 0644
	return []corev1api.Volume{
		{
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

func (c *runManager) getServiceAccountSecretName(ctx *runContext) string {
	if c.testing != nil && c.testing.getServiceAccountSecretNameStub != nil {
		return c.testing.getServiceAccountSecretNameStub(ctx)
	}

	start := time.Now()
	secretName := ctx.serviceAccount.GetHelper().GetServiceAccountSecretNameRepeat()
	elapsed := time.Since(start)
	klog.V(6).Infof("getServiceAccountSecretName for %q took %v", ctx.pipelineRun.String(), elapsed)

	c.metrics.ObserveRetryDurationByType("RunNamespaceServiceAccountSecretCreation", elapsed)
	return secretName
}

func (c *runManager) createTektonTaskRun(ctx *runContext) error {

	if c.testing != nil && c.testing.createTektonTaskRunStub != nil {
		return c.testing.createTektonTaskRunStub(ctx)
	}

	var err error

	copyInt64Ptr := func(ptr *int64) *int64 {
		if ptr != nil {
			v := *ptr
			return &v
		}
		return nil
	}

	namespace := ctx.runNamespace

	tektonTaskRun := tekton.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonTaskRunName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotationPipelineRunKey: ctx.pipelineRun.GetKey(),
			},
		},
		Spec: tekton.TaskRunSpec{
			ServiceAccountName: serviceAccountName,
			TaskRef: &tekton.TaskRef{
				Kind: tekton.ClusterTaskKind,
				Name: tektonClusterTaskName,
			},
			Params: []tekton.Param{
				tektonStringParam("RUN_NAMESPACE", namespace),
			},
			Timeout: ctx.pipelineRunsConfig.Timeout,

			// Always set a non-empty pod template even if we don't have
			// values to set. Otherwise the Tekton default pod template
			// would be used only in such cases but not if we have values
			// to set.
			PodTemplate: &tekton.PodTemplate{
				SecurityContext: &corev1api.PodSecurityContext{
					RunAsUser:  copyInt64Ptr(ctx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsUser),
					RunAsGroup: copyInt64Ptr(ctx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup),
					FSGroup:    copyInt64Ptr(ctx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextFSGroup),
				},
				Volumes: c.volumesWithServiceAccountSecret(ctx),
			},
		},
	}
	c.addTektonTaskRunParamsForJenkinsfileRunnerImage(ctx, &tektonTaskRun)
	err = c.addTektonTaskRunParamsForPipeline(ctx, &tektonTaskRun)
	if err != nil {
		return serrors.Classify(err, stewardv1alpha1.ResultErrorConfig)
	}
	err = c.addTektonTaskRunParamsForLoggingElasticsearch(ctx, &tektonTaskRun)
	if err != nil {
		return serrors.Classify(err, stewardv1alpha1.ResultErrorConfig)
	}

	c.addTektonTaskRunParamsForRunDetails(ctx, &tektonTaskRun)
	tektonClient := c.factory.TektonV1beta1()
	_, err = tektonClient.TaskRuns(tektonTaskRun.GetNamespace()).Create(&tektonTaskRun)
	return err
}

func (c *runManager) addTektonTaskRunParamsForJenkinsfileRunnerImage(
	ctx *runContext,
	tektonTaskRun *tekton.TaskRun,
) {
	spec := ctx.pipelineRun.GetSpec()
	jfrSpec := spec.JenkinsfileRunner
	image := ctx.pipelineRunsConfig.JenkinsfileRunnerImage
	imagePullPolicy := ctx.pipelineRunsConfig.JenkinsfileRunnerImagePullPolicy

	if jfrSpec != nil {
		if jfrSpec.Image != "" {
			image = jfrSpec.Image
			if jfrSpec.ImagePullPolicy == "" {
				imagePullPolicy = "IfNotPresent"
			} else {
				imagePullPolicy = jfrSpec.ImagePullPolicy
			}
		}
	}
	params := []tekton.Param{
		tektonStringParam("JFR_IMAGE", image),
		tektonStringParam("JFR_IMAGE_PULL_POLICY", imagePullPolicy),
	}
	tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)
}

func (c *runManager) addTektonTaskRunParamsForRunDetails(
	ctx *runContext,
	tektonTaskRun *tekton.TaskRun,
) {
	spec := ctx.pipelineRun.GetSpec()
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
		tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)
	}
}

func (c *runManager) addTektonTaskRunParamsForPipeline(
	ctx *runContext,
	tektonTaskRun *tekton.TaskRun,
) error {
	var err error

	spec := ctx.pipelineRun.GetSpec()
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

	tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)
	return nil
}

func (c *runManager) addTektonTaskRunParamsForLoggingElasticsearch(
	ctx *runContext,
	tektonTaskRun *tekton.TaskRun,
) error {
	spec := ctx.pipelineRun.GetSpec()
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

		params = append(params, tektonStringParam("PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON", runIDJSON))
		// use default values from build template for all other params

		if spec.Logging.Elasticsearch.IndexURL != "" {

			_, err := ensureValidElasticsearchIndexURL(spec.Logging.Elasticsearch.IndexURL)
			if err != nil {
				return errors.Wrapf(err,
					"field \"spec.logging.elasticsearch.indexURL\" has invalid value %q",
					spec.Logging.Elasticsearch.IndexURL,
				)
			}
			// use default values from build template for now
		}
	}
	tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)

	return nil
}

// GetRun based on a pipelineRun
func (c *runManager) GetRun(pipelineRun k8s.PipelineRun) (runifc.Run, error) {
	namespace := pipelineRun.GetRunNamespace()
	run, err := c.factory.TektonV1beta1().TaskRuns(namespace).Get(tektonTaskRunName, metav1.GetOptions{})
	if err != nil {
		return nil, serrors.RecoverableIf(err,
			k8serrors.IsServerTimeout(err) ||
				k8serrors.IsServiceUnavailable(err) ||
				k8serrors.IsTimeout(err) ||
				k8serrors.IsTooManyRequests(err) ||
				k8serrors.IsInternalError(err) ||
				k8serrors.IsUnexpectedServerError(err))
	}
	return NewRun(run), nil
}

// Cleanup a run based on a pipelineRun
func (c *runManager) Cleanup(pipelineRun k8s.PipelineRun) error {
	ctx := &runContext{
		pipelineRun:  pipelineRun,
		runNamespace: pipelineRun.GetRunNamespace(),
		auxNamespace: pipelineRun.GetAuxNamespace(),
	}
	return c.cleanupNamespaces(ctx)
}

func (c *runManager) cleanupNamespaces(ctx *runContext) error {
	if c.testing != nil && c.testing.cleanupStub != nil {
		return c.testing.cleanupStub(ctx)
	}

	var deleteOptions *metav1.DeleteOptions
	{
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions = &metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
	}
	errors := []error{}
	namespacesToDelete := []string{
		ctx.runNamespace,
		ctx.auxNamespace,
	}
	for _, name := range namespacesToDelete {
		if name == "" {
			continue
		}
		err := c.deleteNamespace(name, deleteOptions)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) == 0 {
		return nil
	}
	if len(errors) == 1 {
		return errors[0]
	}
	msg := []string{}
	for _, e := range errors {
		msg = append(msg, e.Error())
	}
	return fmt.Errorf("cannot delete all namespaces: %s", strings.Join(msg, ", "))
}

func (c *runManager) createNamespace(ctx *runContext, purpose, randName string) (string, error) {
	var err error

	wanted := &corev1api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-%s-", runNamespacePrefix, randName, purpose),
		},
	}

	slabels.LabelAsSystemManaged(wanted)
	err = slabels.LabelAsOwnedByPipelineRun(wanted, ctx.pipelineRun.GetAPIObject())
	if err != nil {
		return "", errors.Wrap(err, "failed to label namespace as owned by pipeline run")
	}

	isRetriable := func(err error) bool {
		return k8serrors.IsConflict(err) ||
			k8serrors.IsInternalError(err) ||
			k8serrors.IsServerTimeout(err) ||
			k8serrors.IsServiceUnavailable(err) ||
			k8serrors.IsTimeout(err) ||
			k8serrors.IsTooManyRequests(err) ||
			k8serrors.IsUnexpectedServerError(err)
	}

	var created *corev1api.Namespace

	err = retry.OnError(retry.DefaultBackoff, isRetriable,
		func() error {
			var err error
			created, err = c.factory.CoreV1().Namespaces().Create(wanted)
			return err
		},
	)
	if err != nil {
		return "", err
	}

	return created.GetName(), err
}

func (c *runManager) deleteNamespace(name string, options *metav1.DeleteOptions) error {
	isIgnorable := func(err error) bool {
		return k8serrors.IsNotFound(err) ||
			k8serrors.IsGone(err) ||
			k8serrors.IsResourceExpired(err)
	}

	isRetriable := func(err error) bool {
		return k8serrors.IsConflict(err) ||
			k8serrors.IsInternalError(err) ||
			k8serrors.IsServerTimeout(err) ||
			k8serrors.IsServiceUnavailable(err) ||
			k8serrors.IsTimeout(err) ||
			k8serrors.IsTooManyRequests(err) ||
			k8serrors.IsUnexpectedServerError(err)
	}

	return retry.OnError(retry.DefaultBackoff, isRetriable,
		func() error {
			err := c.factory.CoreV1().Namespaces().Delete(name, options)
			if isIgnorable(err) {
				return nil
			}
			return err
		},
	)
}

func toJSONString(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", errors.Wrapf(err, "error while serializing to JSON: %v", err)
	}
	return string(bytes), nil
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

func ensureValidElasticsearchIndexURL(indexURL string) (string, error) {
	validURL, err := url.Parse(indexURL)
	if err != nil {
		return "", err
	}
	if !(strings.ToLower(validURL.Scheme) == "http") && !(strings.ToLower(validURL.Scheme) == "https") {
		return "", fmt.Errorf("scheme not supported: %q", validURL.Scheme)
	}

	return validURL.String(), nil
}
