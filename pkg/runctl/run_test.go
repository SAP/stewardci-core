package runctl

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/ghodss/yaml"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	time1                     = `2019-05-14T08:24:08Z`
	emptyBuild                = `{}`
	startedBuild              = `{"status": {"startTime": "` + time1 + `"}}`
	runningBuild              = `{"status": {"steps": [{"name": "jenkinsfile-runner", "running": {"startedAt": "` + time1 + `"}}]}}`
	completedSuccess          = `{"status": {"conditions": [{"message": "message1", "reason": "Succeeded", "status": "True", "type": "Succeeded"}], "steps": [{"name": "jenkinsfile-runner", "terminated": {"reason": "Completed", "message": "ok", "exitCode": 0}}]}}`
	completedFail             = `{"status": {"conditions": [{"message": "message1", "reason": "Failed", "status": "False", "type": "Succeeded"}], "steps": [{"name": "jenkinsfile-runner", "terminated": {"reason": "Error", "message": "ko", "exitCode": 1}}]}}`
	completedValidationFailed = `{"status": {"conditions": [{"message": "message1", "reason": "TaskRunValidationFailed", "status": "False", "type": "Succeeded"}]}}`
	//See issue https://github.com/SAP/stewardci-core/issues/? TODO: create public issue. internal: 21
	timeout = `{"status": {"conditions": [{"message": "TaskRun \"steward-jenkinsfile-runner\" failed to finish within \"10m0s\"", "reason": "TaskRunTimeout", "status": "False", "type": "Succeeded"}]}}`

	realStartedBuild = `status:
  conditions:
  - lastTransitionTime: "2019-05-14T08:24:12Z"
    message: Not all Steps in the Task have finished executing
    reason: Running
    status: Unknown
    type: Succeeded
  podName: build-pod-38aa76
  startTime: "` + time1 + `"
  steps:
  - container: step-jenkinsfile-runner
    imageID: docker-pullable://alpine@sha256:acd3ca9941a85e8ed16515bfc5328e4e2f8c128caa72959a58a127b7801ee01f
    name: jenkinsfile-runner
    running:
      startedAt: "2019-05-14T08:24:11Z"
`

	realCompletedSuccess = `status:
  completionTime: "2019-05-14T08:24:49Z"
  conditions:
  - lastTransitionTime: "2019-10-04T13:57:28Z"
    message: All Steps have completed executing
    reason: Succeeded
    status: "True"
    type: Succeeded
  podName: build-pod-38aa76
  startTime: "2019-05-14T08:24:08Z"
  steps:
  - container: step-jenkinsfile-runner
    imageID: docker-pullable://alpine@sha256:acd3ca9941a85e8ed16515bfc5328e4e2f8c128caa72959a58a127b7801ee01f
    name: jenkinsfile-runner
    terminated:
      containerID: docker://2ee92b9e6971cd76f896c5c4dc403203754bd4aa6c5191541e5f7d8e04ce9326
      exitCode: 0
      finishedAt: "2019-05-14T08:24:49Z"
      reason: Completed
      startedAt: "2019-05-14T08:24:11Z"
`

	completedMessageSuccess = `status:
  completionTime: "2019-05-14T08:24:49Z"
  conditions:
  - lastTransitionTime: "2019-10-04T13:57:28Z"
    message: All Steps have completed executing
    reason: Succeeded
    status: "True"
    type: Succeeded
  podName: build-pod-38aa76
  startTime: "2019-05-14T08:24:08Z"
  steps:
  - container: step-jenkinsfile-runner
    imageID: docker-pullable://alpine@sha256:acd3ca9941a85e8ed16515bfc5328e4e2f8c128caa72959a58a127b7801ee01f
    name: jenkinsfile-runner
    terminated:
      containerID: docker://2ee92b9e6971cd76f896c5c4dc403203754bd4aa6c5191541e5f7d8e04ce9326
      exitCode: 0
      finishedAt: "2019-05-14T08:24:49Z"
      reason: Completed
      message: %q
      startedAt: "2019-05-14T08:24:11Z"
`
)

func generateTime(timeRFC3339String string) *metav1.Time {
	t, _ := time.Parse(time.RFC3339, timeRFC3339String)
	mt := metav1.NewTime(t)
	return &mt
}

func fakeTektonTaskRun(s string) *tekton.TaskRun {
	var result tekton.TaskRun
	json.Unmarshal([]byte(s), &result)
	return &result
}

func fakeTektonTaskRunYaml(s string) *tekton.TaskRun {
	var result tekton.TaskRun
	yaml.Unmarshal([]byte(s), &result)
	return &result
}

func Test__GetStartTime_UnsetReturnsNil(t *testing.T) {
	run := NewRun(fakeTektonTaskRun(emptyBuild))
	startTime := run.GetStartTime()
	assert.Assert(t, startTime == nil)
}

func Test__GetStartTime_Set(t *testing.T) {
	expectedTime := generateTime(time1)
	run := NewRun(fakeTektonTaskRunYaml(realStartedBuild))
	startTime := run.GetStartTime()
	assert.Assert(t, expectedTime.Equal(startTime), fmt.Sprintf("Expected: %s, Is: %s", expectedTime, startTime))
}

func Test__IsFinished_RunningUpdatesContainer(t *testing.T) {
	run := NewRun(fakeTektonTaskRun(runningBuild))
	finished, _ := run.IsFinished()
	assert.Assert(t, run.GetContainerInfo().Running != nil)
	assert.Assert(t, finished == false)
}

func Test__IsFinished_CompletedSuccess(t *testing.T) {
	build := fakeTektonTaskRunYaml(realCompletedSuccess)
	run := NewRun(build)
	finished, result := run.IsFinished()
	assert.Assert(t, run.GetContainerInfo().Terminated != nil)
	assert.Assert(t, finished == true)
	assert.Equal(t, result, api.ResultSuccess)
}

func Test__IsFinished_CompletedFail(t *testing.T) {
	build := fakeTektonTaskRun(completedFail)
	run := NewRun(build)
	finished, result := run.IsFinished()
	assert.Assert(t, run.GetContainerInfo().Terminated != nil)
	assert.Assert(t, finished == true)
	assert.Equal(t, result, api.ResultErrorContent)
}

func Test__IsFinished_CompletedValidationFail(t *testing.T) {
	build := fakeTektonTaskRun(completedValidationFailed)
	run := NewRun(build)
	finished, result := run.IsFinished()
	assert.Assert(t, finished == true)
	assert.Equal(t, result, api.ResultErrorInfra)
}

func Test__IsFinished_Timeout(t *testing.T) {
	run := NewRun(fakeTektonTaskRun(timeout))
	finished, result := run.IsFinished()
	assert.Assert(t, run.GetContainerInfo() == nil)
	assert.Assert(t, finished == true)
	assert.Equal(t, result, api.ResultTimeout)
}

func Test__GetMessage(t *testing.T) {
	for _, test := range []struct {
		name            string
		inputMessage    string
		expectedMessage string
	}{
		{name: "message_ok",
			inputMessage:    `[{"key":"jfr-termination-log","value":"foo"}]`,
			expectedMessage: "foo",
		},
		{name: "wrong_key",
			inputMessage:    `[{"key":"termination-log","value":"foo"}]`,
			expectedMessage: "internal error",
		},
		{name: "empty message",
			inputMessage:    "",
			expectedMessage: "All Steps have completed executing",
		},
		{name: "multi_key",
			inputMessage:    `[{"key": "foo", "value": "bar"}, {"key":"jfr-termination-log","value":"foo"}]`,
			expectedMessage: "foo",
		},
		{name: "invalid_yaml_message",
			inputMessage:    "{no valid yaml",
			expectedMessage: "{no valid yaml",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			buildString := fmt.Sprintf(completedMessageSuccess, test.inputMessage)
			build := fakeTektonTaskRunYaml(buildString)
			run := NewRun(build)
			result := run.GetMessage()
			assert.Equal(t, test.expectedMessage, result)
		})
	}
}
