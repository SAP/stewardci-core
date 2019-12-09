package v1alpha1

const (
	// StatusReasonFailed indicates that the reason for the status
	// is an unspecified failure.
	StatusReasonFailed = "Failed"

	// StatusReasonDependentResourceState indicates that the reason for the
	// status is the state of another resource controlled by this resource.
	StatusReasonDependentResourceState = "InvalidDependentResource"
)
