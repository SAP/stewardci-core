package runmgr

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/featureflag"
	"github.com/SAP/stewardci-core/pkg/k8s"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/constants"
	runifc "github.com/SAP/stewardci-core/pkg/runctl/run"
	"github.com/SAP/stewardci-core/pkg/runctl/secretmgr"
	slabels "github.com/SAP/stewardci-core/pkg/stewardlabels"
	"github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	tektonPod "github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
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

// TektonRunManager is an implementation of runifc.Manager based on Tekton.
type TektonRunManager struct {
	factory        k8s.ClientFactory
	secretProvider secrets.SecretProvider

	testing *tektonRunManagerTesting
}

// Compiler check for interface compliance
var _ runifc.Manager = (*TektonRunManager)(nil)

type tektonRunManagerTesting struct {
	cleanupStub                               func(context.Context, *runContext) error
	copySecretsToRunNamespaceStub             func(context.Context, *runContext) (string, []string, error)
	createTektonTaskRunStub                   func(context.Context, *runContext) error
	getSecretManagerStub                      func(*runContext) runifc.SecretManager
	prepareRunNamespaceStub                   func(context.Context, *runContext) error
	setupLimitRangeFromConfigStub             func(context.Context, *runContext) error
	setupNetworkPolicyFromConfigStub          func(context.Context, *runContext) error
	setupNetworkPolicyThatIsolatesAllPodsStub func(context.Context, *runContext) error
	setupResourceQuotaFromConfigStub          func(context.Context, *runContext) error
	setupServiceAccountStub                   func(context.Context, *runContext, string, []string) error
	setupStaticLimitRangeStub                 func(context.Context, *runContext) error
	setupStaticNetworkPoliciesStub            func(context.Context, *runContext) error
	setupStaticResourceQuotaStub              func(context.Context, *runContext) error
}

type runContext struct {
	pipelineRun        k8s.PipelineRun
	pipelineRunsConfig *cfg.PipelineRunsConfigStruct
	runNamespace       string
	auxNamespace       string
	serviceAccount     *k8s.ServiceAccountWrap
}

// NewTektonRunManager creates a new TektonRunManager.
func NewTektonRunManager(factory k8s.ClientFactory, secretProvider secrets.SecretProvider) *TektonRunManager {
	return &TektonRunManager{
		factory:        factory,
		secretProvider: secretProvider,
	}
}

// CreateEnv implements runifc.Manager.
func (c *TektonRunManager) CreateEnv(ctx context.Context, pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) (namespace string, auxNamespace string, err error) {

	runCtx := &runContext{
		pipelineRun:        pipelineRun,
		pipelineRunsConfig: pipelineRunsConfig,
		runNamespace:       pipelineRun.GetRunNamespace(),
		auxNamespace:       pipelineRun.GetAuxNamespace(),
	}
	err = c.cleanupNamespaces(ctx, runCtx)
	if err != nil {
		return "", "", err
	}

	// If something goes wrong while creating objects inside the namespaces, we delete everything.
	defer func() {
		if err != nil {
			c.cleanupNamespaces(ctx, runCtx) // clean-up ignoring error
		}
	}()

	err = c.prepareRunNamespace(ctx, runCtx)
	if err != nil {
		return "", "", err
	}

	return runCtx.runNamespace, runCtx.auxNamespace, nil
}

// CreateRun implements runifc.Manager.
func (c *TektonRunManager) CreateRun(ctx context.Context, pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) (err error) {

	runCtx := &runContext{
		pipelineRun:        pipelineRun,
		pipelineRunsConfig: pipelineRunsConfig,
		runNamespace:       pipelineRun.GetRunNamespace(),
		auxNamespace:       pipelineRun.GetAuxNamespace(),
	}

	return c.createTektonTaskRun(ctx, runCtx)
}

// prepareRunNamespace creates a new namespace for the pipeline run
// and populates it with needed resources.
func (c *TektonRunManager) prepareRunNamespace(ctx context.Context, runCtx *runContext) error {

	if c.testing != nil && c.testing.prepareRunNamespaceStub != nil {
		return c.testing.prepareRunNamespaceStub(ctx, runCtx)
	}

	var err error

	randName, err := utils.RandomAlphaNumString(runNamespaceRandomLength)
	if err != nil {
		return err
	}

	runCtx.runNamespace, err = c.createNamespace(ctx, runCtx, "main", randName)
	if err != nil {
		return err
	}

	if featureflag.CreateAuxNamespaceIfUnused.Enabled() {
		runCtx.auxNamespace, err = c.createNamespace(ctx, runCtx, "aux", randName)
		if err != nil {
			return err
		}
	}

	pipelineCloneSecretName, imagePullSecretNames, err := c.copySecretsToRunNamespace(ctx, runCtx)
	if err != nil {
		return err
	}

	err = c.setupServiceAccount(ctx, runCtx, pipelineCloneSecretName, imagePullSecretNames)
	if err != nil {
		return err
	}

	if err = c.setupStaticNetworkPolicies(ctx, runCtx); err != nil {
		return err
	}

	if err = c.setupStaticLimitRange(ctx, runCtx); err != nil {
		return err
	}

	if err = c.setupStaticResourceQuota(ctx, runCtx); err != nil {
		return err
	}

	return nil
}

func (c *TektonRunManager) setupServiceAccount(ctx context.Context, runCtx *runContext, pipelineCloneSecretName string, imagePullSecrets []string) error {
	if c.testing != nil && c.testing.setupServiceAccountStub != nil {
		return c.testing.setupServiceAccountStub(ctx, runCtx, pipelineCloneSecretName, imagePullSecrets)
	}

	accountManager := k8s.NewServiceAccountManager(c.factory, runCtx.runNamespace)
	serviceAccount, err := accountManager.CreateServiceAccount(ctx, serviceAccountName, pipelineCloneSecretName, imagePullSecrets)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "failed to create service account %q", serviceAccountName)
		}
		// service account exists already, so we need to attach secrets to it
		serviceAccount, err = c.attachAllSecrets(ctx, runCtx, accountManager, pipelineCloneSecretName, imagePullSecrets)
		if err != nil {
			return err
		}

	}

	// grant role to service account
	_, err = serviceAccount.AddRoleBinding(ctx, constants.RunClusterRoleName, runCtx.runNamespace)
	if err != nil {
		return errors.Wrapf(err,
			"failed to create role binding for service account %q in namespace %q",
			serviceAccountName, runCtx.runNamespace,
		)
	}

	runCtx.serviceAccount = serviceAccount
	return nil
}

func (c *TektonRunManager) attachAllSecrets(ctx context.Context, runCtx *runContext, accountManager k8s.ServiceAccountManager, pipelineCloneSecretName string, imagePullSecrets []string) (*k8s.ServiceAccountWrap, error) {
	logger := klog.FromContext(ctx)

	var serviceAccount *k8s.ServiceAccountWrap
	for { // retry loop
		var err error
		serviceAccount, err = accountManager.GetServiceAccount(ctx, serviceAccountName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get service account %q", serviceAccountName)
		}

		if pipelineCloneSecretName != "" {
			serviceAccount.AttachSecrets(pipelineCloneSecretName)
		}
		serviceAccount.AttachImagePullSecrets(imagePullSecrets...)
		serviceAccount.SetDoAutomountServiceAccountToken(automountServiceAccountToken)
		err = serviceAccount.Update(ctx)
		if err == nil {
			break // ...the retry loop
		}
		if k8serrors.IsConflict(err) {
			// resource version conflict -> retry update with latest version
			logger.V(4).Info("Retrying service account update",
				"serviceAccount", klog.KRef(runCtx.runNamespace, serviceAccountName),
				"reason", "resource version conflict",
			)
		} else {
			return nil, errors.Wrapf(err, "failed to update service account %q", serviceAccountName)
		}
	}
	return serviceAccount, nil
}

func (c *TektonRunManager) copySecretsToRunNamespace(ctx context.Context, runCtx *runContext) (string, []string, error) {
	if c.testing != nil && c.testing.copySecretsToRunNamespaceStub != nil {
		return c.testing.copySecretsToRunNamespaceStub(ctx, runCtx)
	}
	return c.getSecretManager(runCtx).CopyAll(ctx, runCtx.pipelineRun)
}

func (c *TektonRunManager) getSecretManager(runCtx *runContext) runifc.SecretManager {
	if c.testing != nil && c.testing.getSecretManagerStub != nil {
		return c.testing.getSecretManagerStub(runCtx)
	}
	targetClient := c.factory.CoreV1().Secrets(runCtx.runNamespace)
	secretHelper := secrets.NewSecretHelper(c.secretProvider, runCtx.runNamespace, targetClient)
	return secretmgr.NewSecretManager(secretHelper)
}

func (c *TektonRunManager) setupStaticNetworkPolicies(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupStaticNetworkPoliciesStub != nil {
		return c.testing.setupStaticNetworkPoliciesStub(ctx, runCtx)
	}

	if err := c.setupNetworkPolicyThatIsolatesAllPods(ctx, runCtx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the network policy isolating all pods in namespace %q",
			runCtx.runNamespace,
		)
	}
	if err := c.setupNetworkPolicyFromConfig(ctx, runCtx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured network policy in namespace %q",
			runCtx.runNamespace,
		)
	}
	return nil
}

func (c *TektonRunManager) setupNetworkPolicyThatIsolatesAllPods(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupNetworkPolicyThatIsolatesAllPodsStub != nil {
		return c.testing.setupNetworkPolicyThatIsolatesAllPodsStub(ctx, runCtx)
	}

	policy := &networkingv1api.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: steward.GroupName + "--isolate-all-",
			Namespace:    runCtx.runNamespace,
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

	policyIfce := c.factory.NetworkingV1().NetworkPolicies(runCtx.runNamespace)
	if _, err := policyIfce.Create(ctx, policy, metav1.CreateOptions{}); err != nil {
		return errors.Wrap(err, "error when creating network policy")
	}

	return nil
}

func (c *TektonRunManager) setupNetworkPolicyFromConfig(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupNetworkPolicyFromConfigStub != nil {
		return c.testing.setupNetworkPolicyFromConfigStub(ctx, runCtx)
	}

	networkProfile := runCtx.pipelineRunsConfig.DefaultNetworkProfile

	spec := runCtx.pipelineRun.GetSpec()
	if spec.Profiles != nil && spec.Profiles.Network != "" {
		networkProfile = spec.Profiles.Network

		if _, exists := runCtx.pipelineRunsConfig.NetworkPolicies[networkProfile]; !exists {
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
	manifestYAMLStr := runCtx.pipelineRunsConfig.NetworkPolicies[networkProfile]

	return c.createResource(ctx, manifestYAMLStr, "networkpolicies", "network policy", expectedGroupKind, runCtx)
}

func (c *TektonRunManager) setupStaticLimitRange(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupStaticLimitRangeStub != nil {
		return c.testing.setupStaticLimitRangeStub(ctx, runCtx)
	}

	if err := c.setupLimitRangeFromConfig(ctx, runCtx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured limit range in namespace %q",
			runCtx.runNamespace,
		)
	}

	return nil
}

func (c *TektonRunManager) setupLimitRangeFromConfig(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupLimitRangeFromConfigStub != nil {
		return c.testing.setupLimitRangeFromConfigStub(ctx, runCtx)
	}

	expectedGroupKind := schema.GroupKind{
		Group: "",
		Kind:  "LimitRange",
	}

	configStr := runCtx.pipelineRunsConfig.LimitRange
	if configStr == "" {
		return nil
	}

	return c.createResource(ctx, configStr, "limitranges", "limit range", expectedGroupKind, runCtx)
}

func (c *TektonRunManager) setupStaticResourceQuota(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupStaticResourceQuotaStub != nil {
		return c.testing.setupStaticResourceQuotaStub(ctx, runCtx)
	}

	if err := c.setupResourceQuotaFromConfig(ctx, runCtx); err != nil {
		return errors.Wrapf(err,
			"failed to set up the configured resource quota in namespace %q",
			runCtx.runNamespace,
		)
	}

	return nil
}

func (c *TektonRunManager) setupResourceQuotaFromConfig(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.setupResourceQuotaFromConfigStub != nil {
		return c.testing.setupResourceQuotaFromConfigStub(ctx, runCtx)
	}

	expectedGroupKind := schema.GroupKind{
		Group: "",
		Kind:  "ResourceQuota",
	}

	configStr := runCtx.pipelineRunsConfig.ResourceQuota
	if configStr == "" {
		return nil
	}

	return c.createResource(ctx, configStr, "resourcequotas", "resource quota", expectedGroupKind, runCtx)
}

func (c *TektonRunManager) createResource(ctx context.Context, configStr string, resource string, resourceDisplayName string, expectedGroupKind schema.GroupKind, runCtx *runContext) error {
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
		obj.SetNamespace(runCtx.runNamespace)

		slabels.LabelAsSystemManaged(obj)
	}

	// create resource object
	{
		gvr := schema.GroupVersionResource{
			Group:    expectedGroupKind.Group,
			Version:  obj.GetObjectKind().GroupVersionKind().Version,
			Resource: resource,
		}
		dynamicIfce := c.factory.Dynamic().Resource(gvr).Namespace(runCtx.runNamespace)
		if _, err := dynamicIfce.Create(ctx, obj, metav1.CreateOptions{}); err != nil {
			return errors.Wrapf(err, "failed to create configured %s", resourceDisplayName)
		}
	}

	return nil
}

// GetPipelineRunKeyAnnotation returns the run key of an object
func GetPipelineRunKeyAnnotation(object metav1.Object) string {
	annotations := object.GetAnnotations()
	return annotations[annotationPipelineRunKey]
}

func (c *TektonRunManager) createTektonTaskRunObject(ctx context.Context, runCtx *runContext) (*tekton.TaskRun, error) {

	var err error

	copyInt64Ptr := func(ptr *int64) *int64 {
		if ptr != nil {
			v := *ptr
			return &v
		}
		return nil
	}

	namespace := runCtx.runNamespace
	automount := true
	tektonTaskRun := tekton.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JFRTaskRunName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotationPipelineRunKey: runCtx.pipelineRun.GetKey(),
			},
		},
		Spec: tekton.TaskRunSpec{
			ServiceAccountName: serviceAccountName,

			TaskRef: &tekton.TaskRef{
				ResolverRef: tekton.ResolverRef{
					Resolver: "cluster",
					Params: []tekton.Param{
						tektonStringParam("kind", "task"),
						tektonStringParam("name", runCtx.pipelineRunsConfig.TektonTaskName),
						tektonStringParam("namespace", runCtx.pipelineRunsConfig.TektonTaskNamespace),
					},
				},
			},

			Params: []tekton.Param{
				tektonStringParam("RUN_NAMESPACE", namespace),
			},
			Timeout: getTimeout(runCtx),

			// Always set a non-empty pod template even if we don't have
			// values to set. Otherwise the Tekton default pod template
			// would be used only in such cases but not if we have values
			// to set.
			PodTemplate: &tektonPod.PodTemplate{
				SecurityContext: &corev1api.PodSecurityContext{
					RunAsUser:  copyInt64Ptr(runCtx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsUser),
					RunAsGroup: copyInt64Ptr(runCtx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup),
					FSGroup:    copyInt64Ptr(runCtx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextFSGroup),
				},
				AutomountServiceAccountToken: &automount,
			},
		},
	}
	c.addTektonTaskRunParamsForJenkinsfileRunnerImage(runCtx, &tektonTaskRun)
	err = c.addTektonTaskRunParamsForPipeline(runCtx, &tektonTaskRun)
	if err != nil {
		return nil, serrors.Classify(err, stewardv1alpha1.ResultErrorConfig)
	}
	err = c.addTektonTaskRunParamsForLoggingElasticsearch(runCtx, &tektonTaskRun)
	if err != nil {
		return nil, serrors.Classify(err, stewardv1alpha1.ResultErrorConfig)
	}

	c.addTektonTaskRunParamsForRunDetails(runCtx, &tektonTaskRun)

	return &tektonTaskRun, nil
}

func getTimeout(runCtx *runContext) *metav1.Duration {
	timeout := runCtx.pipelineRunsConfig.Timeout
	pipelineRunTimeout := runCtx.pipelineRun.GetSpec().Timeout
	if pipelineRunTimeout != nil {
		timeout = pipelineRunTimeout
	}
	return timeout
}

func (c *TektonRunManager) createTektonTaskRun(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.createTektonTaskRunStub != nil {
		return c.testing.createTektonTaskRunStub(ctx, runCtx)
	}

	tektonTaskRun, err := c.createTektonTaskRunObject(ctx, runCtx)
	if err != nil {
		return err
	}
	tektonClient := c.factory.TektonV1beta1()
	_, err = tektonClient.TaskRuns(tektonTaskRun.GetNamespace()).Create(ctx, tektonTaskRun, metav1.CreateOptions{})
	return err
}

func (c *TektonRunManager) addTektonTaskRunParamsForJenkinsfileRunnerImage(
	runCtx *runContext,
	tektonTaskRun *tekton.TaskRun,
) {
	spec := runCtx.pipelineRun.GetSpec()
	jfrSpec := spec.JenkinsfileRunner
	image := runCtx.pipelineRunsConfig.JenkinsfileRunnerImage
	imagePullPolicy := runCtx.pipelineRunsConfig.JenkinsfileRunnerImagePullPolicy

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
	params := tekton.Params{
		tektonStringParam("JFR_IMAGE", image),
		tektonStringParam("JFR_IMAGE_PULL_POLICY", imagePullPolicy),
	}
	tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)
}

func (c *TektonRunManager) addTektonTaskRunParamsForRunDetails(
	runCtx *runContext,
	tektonTaskRun *tekton.TaskRun,
) {
	spec := runCtx.pipelineRun.GetSpec()
	details := spec.RunDetails
	if details != nil {
		params := tekton.Params{}
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

func (c *TektonRunManager) addTektonTaskRunParamsForPipeline(
	runCtx *runContext,
	tektonTaskRun *tekton.TaskRun,
) error {
	var err error

	spec := runCtx.pipelineRun.GetSpec()
	pipeline := spec.JenkinsFile
	pipelineArgs := spec.Args
	pipelineArgsJSON := "{}"
	if pipelineArgs != nil {
		if pipelineArgsJSON, err = utils.ToJSONString(&pipelineArgs); err != nil {
			return err
		}
	}

	params := tekton.Params{
		tektonStringParam("PIPELINE_GIT_URL", pipeline.URL),
		tektonStringParam("PIPELINE_GIT_REVISION", pipeline.Revision),
		tektonStringParam("PIPELINE_FILE", pipeline.Path),
		tektonStringParam("PIPELINE_PARAMS_JSON", pipelineArgsJSON),
	}

	tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)
	return nil
}

func (c *TektonRunManager) addTektonTaskRunParamsForLoggingElasticsearch(
	runCtx *runContext,
	tektonTaskRun *tekton.TaskRun,
) error {
	spec := runCtx.pipelineRun.GetSpec()
	var params tekton.Params

	if spec.Logging == nil || spec.Logging.Elasticsearch == nil {
		params = tekton.Params{
			// overide the index URL hardcoded in the template by
			// the empty string to effective disable logging to
			// Elasticsearch
			tektonStringParam("PIPELINE_LOG_ELASTICSEARCH_INDEX_URL", ""),
		}
	} else {
		runIDJSON, err := utils.ToJSONString(&spec.Logging.Elasticsearch.RunID)
		if err != nil {
			return errors.WithMessage(err,
				"could not serialize spec.logging.elasticsearch.runid to JSON",
			)
		}

		params = append(params, tektonStringParam("PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON", runIDJSON))
		// use default values from build template for all other params

		if spec.Logging.Elasticsearch.IndexURL != "" {

			validURL, err := ensureValidElasticsearchIndexURL(spec.Logging.Elasticsearch.IndexURL)
			if err != nil {
				return errors.Wrapf(err,
					"field \"spec.logging.elasticsearch.indexURL\" has invalid value %q",
					spec.Logging.Elasticsearch.IndexURL,
				)
			}
			params = append(params, tektonStringParam("PIPELINE_LOG_ELASTICSEARCH_INDEX_URL", validURL))
		}
	}
	tektonTaskRun.Spec.Params = append(tektonTaskRun.Spec.Params, params...)

	return nil
}

func (c *TektonRunManager) recoverableIfTransient(err error) error {
	return serrors.RecoverableIf(err,
		k8serrors.IsServerTimeout(err) ||
			k8serrors.IsServiceUnavailable(err) ||
			k8serrors.IsTimeout(err) ||
			k8serrors.IsTooManyRequests(err) ||
			k8serrors.IsInternalError(err) ||
			k8serrors.IsUnexpectedServerError(err))
}

// GetRun creates a new TektonRunManager.
func (c *TektonRunManager) GetRun(ctx context.Context, pipelineRun k8s.PipelineRun) (runifc.Run, error) {
	namespace := pipelineRun.GetRunNamespace()
	run, err := c.factory.TektonV1beta1().TaskRuns(namespace).Get(ctx, JFRTaskRunName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, c.recoverableIfTransient(err)
	}
	return newRun(run), nil
}

// DeleteRun creates a new TektonRunManager.
func (c *TektonRunManager) DeleteRun(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	namespace := pipelineRun.GetRunNamespace()
	if namespace == "" {
		return fmt.Errorf("cannot delete taskrun, run namespace not set in %q", pipelineRun.GetName())
	}
	err := c.factory.TektonV1beta1().TaskRuns(namespace).Delete(ctx, JFRTaskRunName, metav1.DeleteOptions{})

	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return c.recoverableIfTransient(err)
	}
	return nil
}

// DeleteEnv creates a new TektonRunManager.
func (c *TektonRunManager) DeleteEnv(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	runCtx := &runContext{
		pipelineRun:  pipelineRun,
		runNamespace: pipelineRun.GetRunNamespace(),
		auxNamespace: pipelineRun.GetAuxNamespace(),
	}
	return c.cleanupNamespaces(ctx, runCtx)
}

func (c *TektonRunManager) cleanupNamespaces(ctx context.Context, runCtx *runContext) error {
	if c.testing != nil && c.testing.cleanupStub != nil {
		return c.testing.cleanupStub(ctx, runCtx)
	}

	var deleteOptions metav1.DeleteOptions
	{
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions = metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
	}
	errors := []error{}
	namespacesToDelete := []string{
		runCtx.runNamespace,
		runCtx.auxNamespace,
	}
	for _, name := range namespacesToDelete {
		if name == "" {
			continue
		}
		err := c.deleteNamespace(ctx, name, deleteOptions)
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

func (c *TektonRunManager) createNamespace(ctx context.Context, runCtx *runContext, purpose, randName string) (string, error) {
	var err error

	wanted := &corev1api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-%s-", runNamespacePrefix, randName, purpose),
		},
	}

	slabels.LabelAsSystemManaged(wanted)
	err = slabels.LabelAsOwnedByPipelineRun(wanted, runCtx.pipelineRun.GetAPIObject())
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
			created, err = c.factory.CoreV1().Namespaces().Create(ctx, wanted, metav1.CreateOptions{})
			return err
		},
	)
	if err != nil {
		return "", err
	}

	return created.GetName(), err
}

func (c *TektonRunManager) deleteNamespace(ctx context.Context, name string, options metav1.DeleteOptions) error {
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
			err := c.factory.CoreV1().Namespaces().Delete(ctx, name, options)
			if isIgnorable(err) {
				return nil
			}
			return err
		},
	)
}

func tektonStringParam(name string, value string) tekton.Param {
	return tekton.Param{
		Name: name,
		Value: tekton.ParamValue{
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
	if strings.ToLower(validURL.Scheme) != "http" && strings.ToLower(validURL.Scheme) != "https" {
		return "", fmt.Errorf("scheme not supported: %q", validURL.Scheme)
	}

	return validURL.String(), nil
}
