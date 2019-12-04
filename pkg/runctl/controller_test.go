package runctl

import (
	"fmt"
	"log"
	"strings"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	metrics "github.com/SAP/stewardci-core/pkg/metrics"
	gomock "github.com/golang/mock/gomock"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	assert "gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

func Test_Controller_MissingSecret(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory()
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})
	// no "secret1" here

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY

	run, err := getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	status := run.GetStatus()

	assert.Equal(t, api.StateFinished, status.State)
	assert.Equal(t, api.ResultErrorContent, status.Result)
	//TODO: namespace is deleted twice, second fails. We need to check why and make sure the correct error is in the message.
	// MR: namespaceManager changed to return nil error if not existing ns is deleted
	assert.Assert(t, is.Regexp("failed to copy secrets: .*", status.Message))
}

func Test_Controller_Success(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		fake.ClusterRole(string(runClusterRoleName)),
	)
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY
	run, err := getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	status := run.GetStatus()

	assert.Assert(t, !strings.Contains(status.Message, "ERROR"), status.Message)
	assert.Equal(t, api.StateWaiting, status.State)
	assert.Equal(t, 2, len(status.StateHistory))
}

func Test_Controller_Running(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		fake.ClusterRole(string(runClusterRoleName)),
	)
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY
	run, err := getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	runNs := run.GetRunNamespace()
	taskRun, _ := getTektonTaskRun(runNs, cf)
	now := metav1.Now()
	taskRun.Status.StartTime = &now
	updateTektonTaskRun(taskRun, runNs, cf)
	cf.Sleep("Waiting for Tekton TaskRun being started")
	run, err = getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	status := run.GetStatus()
	assert.Equal(t, api.StateRunning, status.State)
}

func Test_Controller_Deletion(t *testing.T) {
	t.Parallel()
	// SETUP
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})
	cf := fake.NewClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		fake.ClusterRole(string(runClusterRoleName)),
	)

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY
	run, _ := getRun("run1", "ns1", cf)

	assert.Equal(t, 1, len(run.GetFinalizers()))

	now := metav1.Now()
	run.SetDeletionTimestamp(&now)
	updateRun(run, "ns1", cf)

	cf.Sleep("Wait for deletion")
	run, _ = getRun("run1", "ns1", cf)
	assert.Equal(t, 0, len(run.GetFinalizers()))

}

func Test_Controller_syncHandler_givesUp_onPipelineRunNotFound(t *testing.T) {
	t.Parallel()
	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := fake.NewClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	mockPipelineRunFetcher.EXPECT().
		ByKey(gomock.Any()).
		Return(nil, nil)
	examinee := NewController(cf, metrics.NewMetrics())
	examinee.pipelineRunFetcher = mockPipelineRunFetcher

	// EXERCISE
	err := examinee.syncHandler("foo/bar")
	// VERIFY
	assert.NilError(t, err)
}

func Test_Controller_syncHandler_initiatesRetrying_on500DuringPipelineRunFetch(t *testing.T) {
	t.Parallel()
	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := fake.NewClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	message := "k8s kapot!"
	mockPipelineRunFetcher.EXPECT().
		ByKey(gomock.Any()).
		Return(nil, k8serrors.NewInternalError(fmt.Errorf(message)))

	examinee := NewController(cf, metrics.NewMetrics())
	examinee.pipelineRunFetcher = mockPipelineRunFetcher
	// EXERCISE
	err := examinee.syncHandler("foo/bar")
	// VERIFY
	assert.ErrorContains(t, err, message)
}

func Test_Controller_syncHandler_OnTimeout(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory(

		// the tenant namespace
		fake.Namespace("tenant-ns-1"),

		// the Steward PipelineRun in status running
		StewardObjectFromJSON(t, `{
			"apiVersion": "steward.sap.com/v1alpha1",
			"kind": "PipelineRun",
			"metadata": {
				"name": "run1",
				"namespace": "tenant-ns-1",
				"uid": "a9e79ee8-69a8-4d8b-8a29-f51b53ada9b7"
			},
			"spec": {},
			"status": {
				"namespace": "steward-run-ns-1",
				"state": "running"
			}
		}`),

		// the run namespace
		// label is required for deletion
		CoreV1ObjectFromJSON(t, `{
			"apiVersion": "v1",
			"kind": "Namespace",
			"metadata": {
				"name": "steward-run-ns-1",
				"labels": {
					"id": "tenant1",
					"prefix": "steward-run"
				}
			}
		}`),

		// the Tekton TaskRun
		TektonObjectFromJSON(t, `{
			"apiVersion": "tekton.dev/v1alpha1",
			"kind": "TaskRun",
			"metadata": {
				"name": "steward-jenkinsfile-runner",
				"namespace": "steward-run-ns-1"
			},
			"spec": {},
			"status": {
				"conditions": [
					{
						"lastTransitionTime": "2019-09-16T12:55:40Z",
						"message": "message from Succeeded condition",
						"reason": "TaskRunTimeout",
						"status": "False",
						"type": "Succeeded"
					}
				],
				"startTime": "2019-09-16T12:45:40Z",
				"completionTime": "2019-09-16T12:55:40Z"
			}
		}`),

		fake.ClusterRole(string(runClusterRoleName)),
	)

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)

	// VERIFY
	run, err := getPipelineRun("run1", "tenant-ns-1", cf)
	assert.NilError(t, err)

	status := run.GetStatus()

	assert.Assert(t, status != nil)
	assert.Equal(t, api.StateFinished, status.State)
	assert.Equal(t, status.State, status.StateDetails.State)
	assert.Equal(t, api.ResultTimeout, status.Result)
	assert.Equal(t, "message from Succeeded condition", status.Message)
}

func startController(t *testing.T, cf *fake.ClientFactory) chan struct{} {
	cs := cf.StewardClientset()
	cs.PrependReactor("create", "*", fake.NewCreationTimestampReactor())
	stopCh := make(chan struct{}, 0)
	metrics := metrics.NewMetrics()
	controller := NewController(cf, metrics)
	controller.pipelineRunFetcher = k8s.NewClientBasedPipelineRunFetcher(cf.StewardV1alpha1())
	cf.StewardInformerFactory().Start(stopCh)
	cf.TektonInformerFactory().Start(stopCh)
	go start(t, controller, stopCh)
	cf.Sleep("Wait for controller")
	return stopCh
}

func stopController(t *testing.T, stopCh chan struct{}) {
	log.Printf("Trigger controller stop")
	stopCh <- struct{}{}
}

func start(t *testing.T, controller *Controller, stopCh chan struct{}) {
	if err := controller.Run(1, stopCh); err != nil {
		t.Logf("Error running controller %s", err.Error())
	}
}

func resource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: "", Resource: resource}
}

// GetPipelineRun returns the pipeline run with the given name in the given namespace.
// Return nil if not found.
func getPipelineRun(name string, namespace string, cf *fake.ClientFactory) (k8s.PipelineRun, error) {
	key := fake.ObjectKey(name, namespace)
	fetcher := k8s.NewClientBasedPipelineRunFetcher(cf.StewardV1alpha1())
	pipelineRun, err := fetcher.ByKey(key)
	return k8s.NewPipelineRun(pipelineRun, fetcher, cf), err
}

func createRun(run *api.PipelineRun, cf *fake.ClientFactory) error {
	_, err := cf.StewardV1alpha1().PipelineRuns(run.GetNamespace()).Create(run)
	if err == nil {
		cf.Sleep("wait for controller to pick up run")
	}
	return err
}

func getRun(name, namespace string, cf *fake.ClientFactory) (*api.PipelineRun, error) {
	return cf.StewardV1alpha1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})
}

func updateRun(run *api.PipelineRun, namespace string, cf *fake.ClientFactory) (*api.PipelineRun, error) {
	return cf.StewardV1alpha1().PipelineRuns(namespace).Update(run)
}

func getTektonTaskRun(namespace string, cf *fake.ClientFactory) (*tekton.TaskRun, error) {
	return cf.TektonV1alpha1().TaskRuns(namespace).Get(tektonTaskRunName, metav1.GetOptions{})
}

func updateTektonTaskRun(taskRun *tekton.TaskRun, namespace string, cf *fake.ClientFactory) (*tekton.TaskRun, error) {
	return cf.TektonV1alpha1().TaskRuns(namespace).Update(taskRun)
}
