/*
#########################
#  SAP Steward-CI       #
#########################

THIS CODE IS GENERATED! DO NOT TOUCH!

Copyright SAP SE.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// PipelineRunDetailsApplyConfiguration represents an declarative configuration of the PipelineRunDetails type for use
// with apply.
type PipelineRunDetailsApplyConfiguration struct {
	JobName        *string `json:"jobName,omitempty"`
	SequenceNumber *int32  `json:"sequenceNumber,omitempty"`
	Cause          *string `json:"cause,omitempty"`
}

// PipelineRunDetailsApplyConfiguration constructs an declarative configuration of the PipelineRunDetails type for use with
// apply.
func PipelineRunDetails() *PipelineRunDetailsApplyConfiguration {
	return &PipelineRunDetailsApplyConfiguration{}
}

// WithJobName sets the JobName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the JobName field is set to the value of the last call.
func (b *PipelineRunDetailsApplyConfiguration) WithJobName(value string) *PipelineRunDetailsApplyConfiguration {
	b.JobName = &value
	return b
}

// WithSequenceNumber sets the SequenceNumber field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SequenceNumber field is set to the value of the last call.
func (b *PipelineRunDetailsApplyConfiguration) WithSequenceNumber(value int32) *PipelineRunDetailsApplyConfiguration {
	b.SequenceNumber = &value
	return b
}

// WithCause sets the Cause field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Cause field is set to the value of the last call.
func (b *PipelineRunDetailsApplyConfiguration) WithCause(value string) *PipelineRunDetailsApplyConfiguration {
	b.Cause = &value
	return b
}
