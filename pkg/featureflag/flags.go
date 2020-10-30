package featureflag

var (
	// Dummy shows how to define a feature flag. DO NOT USE IT!
	Dummy = New("Dummy", Bool(false))

	// RetryOnInvalidPipelineRunsConfig controls whether the execution of a pipeline run
	// is failed or retried on pipeline run configuration errors.
	RetryOnInvalidPipelineRunsConfig = New("RetryOnInvalidPipelineRunsConfig", Bool(false))
)
