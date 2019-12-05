package k8s

import (
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	stewardLister "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// PipelineRunFetcher combines PipelineRunByKeyFetcher and PipelineRunByNameFetcher
type PipelineRunFetcher interface {
	PipelineRunByKeyFetcher
	PipelineRunByNameFetcher
}

// PipelineRunByKeyFetcher provides a function to fetch PipelineRuns by their key
type PipelineRunByKeyFetcher interface {
	// ByKey fetches PipelineRun resource from Kubernetes
	// Return nil,nil if pipeline with key does not exist
	ByKey(key string) (*api.PipelineRun, error)
}

// PipelineRunByNameFetcher provides a function to fetch PipelineRuns by their name
type PipelineRunByNameFetcher interface {
	// ByName fetches PipelineRun resource from Kubernetes by name and namespace
	// Return nil,nil if specified pipeline does not exist
	ByName(namespace, name string) (*api.PipelineRun, error)
}

type listerBasedPipelineRunFetcher struct {
	lister stewardLister.PipelineRunLister
}

// NewListerBasedPipelineRunFetcher returns a PipelineRunFetcher that retrieves
// the objects from the given `PipelineRunLister`.
// The returned fetcher provides the original pointers from the lister. Typically the lister
// is backed by a shared cache which must not be modified. Consumers should not
// mutate the original objects, but create deep copies when modification is required.
func NewListerBasedPipelineRunFetcher(lister stewardLister.PipelineRunLister) PipelineRunFetcher {
	return &listerBasedPipelineRunFetcher{
		lister: lister,
	}
}

// ByName implements interface PipelineRunByNameFetcher
func (f *listerBasedPipelineRunFetcher) ByName(namespace, name string) (*api.PipelineRun, error) {
	lister := f.lister.PipelineRuns(namespace)
	return returnCopyOrNilOnNotFound(lister.Get(name))
}

// ByKey implements interface PipelineRunByKeyFetcher
func (f *listerBasedPipelineRunFetcher) ByKey(key string) (*api.PipelineRun, error) {
	return byKey(f, key)
}

type clientBasedPipelineRunFetcher struct {
	client stewardv1alpha1.StewardV1alpha1Interface
}

// NewClientBasedPipelineRunFetcher returns a PipelineRunFetcher that retrieves
// the objects from the given API client.
func NewClientBasedPipelineRunFetcher(client stewardv1alpha1.StewardV1alpha1Interface) PipelineRunFetcher {
	return &clientBasedPipelineRunFetcher{client: client}
}

// ByName implements interface PipelineRunByNameFetcher
func (rf *clientBasedPipelineRunFetcher) ByName(namespace string, name string) (*api.PipelineRun, error) {
	client := rf.client.PipelineRuns(namespace)
	run, err := client.Get(name, metav1.GetOptions{})
	return returnCopyOrNilOnNotFound(run, err)
}

// ByKey implements interface PipelineRunByKeyFetcher
func (rf *clientBasedPipelineRunFetcher) ByKey(key string) (*api.PipelineRun, error) {
	return byKey(rf, key)
}

func byKey(rf PipelineRunByNameFetcher, key string) (*api.PipelineRun, error) {
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
