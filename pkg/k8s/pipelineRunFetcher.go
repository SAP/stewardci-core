package k8s

import (
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardLister "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// PipelineRunFetcher has methods to fetch PipelineRun objects from Kubernetes
type PipelineRunFetcher interface {
	pipelineRunNameFetcher
	ByKey(key string) (*api.PipelineRun, error)
}

type pipelineRunNameFetcher interface {
	ByName(namespace, name string) (*api.PipelineRun, error)
}

type pipelineRunListerFetcher struct {
	lister stewardLister.PipelineRunLister
}

// NewPipelineRunListerFetcher returns an operative implementation of PipelineRunFetcher
func NewPipelineRunListerFetcher(lister stewardLister.PipelineRunLister) PipelineRunFetcher {
	return &pipelineRunListerFetcher{
		lister: lister,
	}
}

// ByName fetches PipelineRun resource from Kubernetes by name and namespace
// Return nil,nil if specified pipeline does not exist
func (f *pipelineRunListerFetcher) ByName(namespace, name string) (*api.PipelineRun, error) {
	lister := f.lister.PipelineRuns(namespace)
	return returnCopyOrNilOnNotFound(lister.Get(name))
}

// ByKey fetches PipelineRun resource from Kubernetes
// Return nil,nil if pipeline with key does not exist
func (f *pipelineRunListerFetcher) ByKey(key string) (*api.PipelineRun, error) {
	return byKey(f, key)
}

type pipelineRunFetcher struct {
	factory ClientFactory
}

// NewPipelineRunFetcher returns an operative implementation of PipelineRunFetcher
func NewPipelineRunFetcher(factory ClientFactory) PipelineRunFetcher {
	return &pipelineRunFetcher{factory: factory}
}

// ByName fetches PipelineRun resource from Kubernetes by name and namespace
// Return nil,nil if specified pipeline does not exist
func (rf *pipelineRunFetcher) ByName(namespace string, name string) (*api.PipelineRun, error) {
	client := rf.factory.StewardV1alpha1().PipelineRuns(namespace)
	return returnCopyOrNilOnNotFound(client.Get(name, metav1.GetOptions{}))
}

// ByKey fetches PipelineRun resource from Kubernetes
// Return nil,nil if pipeline with key does not exist
func (rf *pipelineRunFetcher) ByKey(key string) (*api.PipelineRun, error) {
	return byKey(rf, key)
}

func byKey(rf pipelineRunNameFetcher, key string) (*api.PipelineRun, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	return rf.ByName(namespace, name)
}

func returnCopyOrNilOnNotFound(run *api.PipelineRun, err error) (*api.PipelineRun, error) {
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err,
			fmt.Sprintf("Failed to fetch PipelineRun '%s' in namespace '%s'", run.GetName(), run.GetNamespace()))
	}
	return run.DeepCopy(), err
}
