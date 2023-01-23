package featureflag

var (
	// Dummy shows how to define a feature flag. DO NOT USE IT!
	Dummy = New("Dummy", Bool(false))

	// CreateAuxNamespaceIfUnused controls whether auxiliary namespaces for
	// pipeline runs are created although they are not used.
	CreateAuxNamespaceIfUnused = New("CreateAuxNamespaceIfUnused", Bool(false))

	// RetryOnInvalidPipelineRunsConfig controls whether the execution of a pipeline run
	// is failed or retried on pipeline run configuration errors.
	RetryOnInvalidPipelineRunsConfig = New("RetryOnInvalidPipelineRunsConfig", Bool(false))

	// CreateServiceAccountTokenInRunNamespace controls whether steward is requesting
	// Service Account Tokens for Run Namespaces. This is required for K8S 1.24+
	CreateServiceAccountTokenInRunNamespace = New("CreateServiceAccountTokenInRunNamespace", Bool(false))
)
