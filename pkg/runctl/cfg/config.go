package cfg

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/featureflag"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	mainConfigMapName                = "steward-pipelineruns"
	mainConfigKeyTimeout             = "timeout"
	mainConfigKeyTimeoutWait         = "waitTimeout"
	mainConfigKeyLimitRange          = "limitRange"
	mainConfigKeyResourceQuota       = "resourceQuota"
	mainConfigKeyLabelsToLog         = "labelsToLog"
	mainConfigKeyImage               = "jenkinsfileRunner.image"
	mainConfigKeyImagePullPolicy     = "jenkinsfileRunner.imagePullPolicy"
	mainConfigKeyPSCRunAsUser        = "jenkinsfileRunner.podSecurityContext.runAsUser"
	mainConfigKeyPSCRunAsGroup       = "jenkinsfileRunner.podSecurityContext.runAsGroup"
	mainConfigKeyPSCFSGroup          = "jenkinsfileRunner.podSecurityContext.fsGroup"
	mainConfigKeyTektonTaskName      = "tektonTaskName"
	mainConfigKeyTektonTaskNamespace = "tektonTaskNamespace"

	networkPoliciesConfigMapName    = "steward-pipelineruns-network-policies"
	networkPoliciesConfigKeyDefault = "_default"
)

// PipelineRunsConfigStruct is a struct holding the pipeline runs configuration.
type PipelineRunsConfigStruct struct {
	// Timeout is the maximum execution time of a pipeline run.
	// If `nil`, a default timeout should be used.
	Timeout *metav1.Duration

	// TimeoutWait is the maximum time a pipeline run can stay in state waiting
	// before it is stopped with timeout error.
	// If `nil`, the timeout is set to 10 minutes.
	TimeoutWait *metav1.Duration

	// The manifest (in YAML format) of a Kubernetes LimitRange object to be
	// applied to each pipeline run sandbox namespace.
	// If empty, no limit range will be defined.
	LimitRange string

	// The manifest (in YAML format) of a Kubernetes ResourceQuota object to be
	// applied to each pipeline run sandbox namespace.
	// If empty, no resource quota will be defined.
	ResourceQuota string

	// LabelsToLog contains a map specifying the values of which labels should be logged
	// in the structured logging under which key.
	LabelsToLog map[string]string

	// JenkinsfileRunnerImage is the Jenkinsfile Runner container image to be
	// used for pipeline runs.
	// If empty, a default image will be used.
	JenkinsfileRunnerImage string

	// JenkinsfileRunnerImagePullPolicy is the pull policy for the container
	// image defined by `JenkinsfileRunnerImage`.
	// It defaults to `IfNotPresent`.
	// If `JenkinsfileRunnerImage` is not set, this value is not used (it does
	// not apply to the default image).
	JenkinsfileRunnerImagePullPolicy string

	// JenkinsfileRunnerPodSecurityContextRunAsUser is the numerical user id
	// the Jenkinsfile Runner process is started as.
	JenkinsfileRunnerPodSecurityContextRunAsUser *int64

	// JenkinsfileRunnerPodSecurityContextRunAsGroup is the numerical group id
	// the Jenkinsfile Runner process is started as.
	JenkinsfileRunnerPodSecurityContextRunAsGroup *int64

	// JenkinsfileRunnerPodSecurityContextFSGroup is the numerical filesystem
	// group id the Jenkinsfile Runner pod will use.
	JenkinsfileRunnerPodSecurityContextFSGroup *int64

	// DefaultNetworkProfile is the name of the network profile that should
	// be used in case the user has not explicitly chosen one.
	DefaultNetworkProfile string

	// NetworkPolicies maps network profile names to network policies.
	// Each value is a Kubernetes network policy manifest in YAML format.
	NetworkPolicies map[string]string

	// TektonTaskName is the name of the Tekton task to run the
	// Jenkinsfile Runner pod.
	TektonTaskName string

	// TektonTaskNamespace is the name of the namespace containing
	// the Tekton task to run the Jenkinsfile Runner.
	TektonTaskNamespace string
}

type configDataMap map[string]string

func (cd configDataMap) parseInt64(key string) (*int64, error) {
	if strVal, ok := cd[key]; ok && strVal != "" {
		intVal, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return nil, wrapParseError(err, key, strVal)
		}
		return &intVal, nil
	}
	return nil, nil
}

func (cd configDataMap) parseDuration(key string) (*metav1.Duration, error) {
	if strVal, ok := cd[key]; ok && strVal != "" {
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return nil, wrapParseError(err, key, strVal)
		}
		return &metav1.Duration{Duration: d}, nil
	}
	return nil, nil
}

func (cd configDataMap) parseMap(key string) (map[string]string, error) {
	result := map[string]string{}
	var err error
	if strVal, ok := cd[key]; ok && strVal != "" {
		err = yaml.Unmarshal([]byte(strVal), &result)
	}
	return result, err
}

// LoadPipelineRunsConfig loads the pipeline run's configuration and returns it.
func LoadPipelineRunsConfig(ctx context.Context, clientFactory k8s.ClientFactory) (*PipelineRunsConfigStruct, error) {
	dest := &PipelineRunsConfigStruct{}

	for _, p := range []struct {
		configMapName string
		optional      bool
		processFunc   func(configDataMap, *PipelineRunsConfigStruct) error
	}{
		{
			configMapName: mainConfigMapName,
			optional:      true,
			processFunc:   processMainConfig,
		},
		{
			configMapName: networkPoliciesConfigMapName,
			optional:      false,
			processFunc:   processNetworkPoliciesConfig,
		},
	} {
		err := processConfigMap(
			ctx,
			p.configMapName, p.optional, p.processFunc,
			dest, clientFactory,
		)
		if err != nil {
			return nil, err
		}
	}

	return dest, nil
}

func withRecoverability(err error, isInfraError bool) error {
	return serrors.RecoverableIf(err, isInfraError || featureflag.RetryOnInvalidPipelineRunsConfig.Enabled())
}

/*
processConfigMap is a higher-order function which calls `processFunc` to
process the config map with the given name and enriches error messages
with contextual information.
`optional` indicated whether the config map may not exist, in which case
`processFunc` is NOT called and NO error is returned.
`dest` is the destination struct to store loaded configuration values in.
It gets passed to `processFunc`.
*/
func processConfigMap(
	ctx context.Context,
	configMapName string,
	optional bool,
	processFunc func(configDataMap, *PipelineRunsConfigStruct) error,
	dest *PipelineRunsConfigStruct,
	clientFactory k8s.ClientFactory,
) error {

	wrapError := func(cause error) error {
		return errors.Wrapf(cause,
			"invalid configuration: ConfigMap %q in namespace %q",
			configMapName,
			system.Namespace(),
		)
	}

	configMapIfce := clientFactory.CoreV1().ConfigMaps(system.Namespace())

	var err error
	configMap, err := configMapIfce.Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return withRecoverability(wrapError(err), true)
	}

	if configMap != nil {
		err = processFunc(configMap.Data, dest)
		if err != nil {
			return withRecoverability(wrapError(err), false)
		}
	} else if !optional {
		return withRecoverability(wrapError(errors.New("is missing")), false)
	}

	return nil
}

func wrapParseError(cause error, key, strVal string) error {
	return errors.Wrapf(cause,
		"key %q: cannot parse value %q",
		key, strVal,
	)
}

func processMainConfig(configData configDataMap, dest *PipelineRunsConfigStruct) error {

	dest.LimitRange = configData[mainConfigKeyLimitRange]
	dest.ResourceQuota = configData[mainConfigKeyResourceQuota]
	dest.JenkinsfileRunnerImage = configData[mainConfigKeyImage]
	dest.JenkinsfileRunnerImagePullPolicy = configData[mainConfigKeyImagePullPolicy]
	dest.TektonTaskName = configData[mainConfigKeyTektonTaskName]
	dest.TektonTaskNamespace = configData[mainConfigKeyTektonTaskNamespace]

	var err error

	if dest.LabelsToLog, err =
		configData.parseMap(mainConfigKeyLabelsToLog); err != nil {
		return err
	}

	if dest.Timeout, err =
		configData.parseDuration(mainConfigKeyTimeout); err != nil {
		return err
	}

	if dest.TimeoutWait, err =
		configData.parseDuration(mainConfigKeyTimeoutWait); err != nil {
		return err
	}

	if dest.JenkinsfileRunnerPodSecurityContextRunAsUser, err =
		configData.parseInt64(mainConfigKeyPSCRunAsUser); err != nil {
		return err
	}
	if dest.JenkinsfileRunnerPodSecurityContextRunAsGroup, err =
		configData.parseInt64(mainConfigKeyPSCRunAsGroup); err != nil {
		return err
	}

	if dest.JenkinsfileRunnerPodSecurityContextFSGroup, err =
		configData.parseInt64(mainConfigKeyPSCFSGroup); err != nil {
		return err
	}

	return nil
}

func processNetworkPoliciesConfig(configData configDataMap, dest *PipelineRunsConfigStruct) error {

	isValidKey := func(key string) bool {
		return key != "" && key == strings.TrimSpace(key) && !strings.HasPrefix(key, "_")
	}

	dest.DefaultNetworkProfile = ""
	dest.NetworkPolicies = nil

	networkPolicies := map[string]string{}
	for key, value := range configData {
		if isValidKey(key) && strings.TrimSpace(value) != "" {
			networkPolicies[key] = value
		}
	}

	var (
		defaultNetworkPolicyKey string
		found                   bool
	)

	if defaultNetworkPolicyKey, found = configData[networkPoliciesConfigKeyDefault]; !found {
		return fmt.Errorf(
			"key %q is missing",
			networkPoliciesConfigKeyDefault,
		)
	}

	if !isValidKey(defaultNetworkPolicyKey) {
		return fmt.Errorf(
			"key %q: value %q is not a valid network policy key",
			networkPoliciesConfigKeyDefault,
			defaultNetworkPolicyKey,
		)
	}

	if _, found = networkPolicies[defaultNetworkPolicyKey]; !found {
		return fmt.Errorf(
			"key %q: value %q does not denote an existing network policy key",
			networkPoliciesConfigKeyDefault,
			defaultNetworkPolicyKey,
		)
	}

	dest.DefaultNetworkProfile = defaultNetworkPolicyKey
	dest.NetworkPolicies = networkPolicies

	return nil
}
