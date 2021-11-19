package workqueue

import (
	"fmt"
	"sync"
)

var (
	nameProvidersInstance *nameProviders = &nameProviders{}
)

// NameProvider is the interface for name providers contributed by
// packages that make use of k8s.io/client-go/util/workqueue.
type NameProvider interface {
	/*
		Returns the subsystem name to be used for Prometheus workqueue metrics for
		the given queue name. If the given queue name is not identified to belong to
		the name provider's package, return value `ok` must be `false`.
	*/
	GetSubsystemFor(queueName string) (subsystem string, ok bool)
}

// NameProviderFunc is an adapter that allows to use functions with the right
// signature as NameProvider.
type NameProviderFunc func(queueName string) (subsystem string, ok bool)

// GetSubsystemFor implements interface NameProvider.
func (f NameProviderFunc) GetSubsystemFor(queueName string) (string, bool) {
	return f(queueName)
}

// nameProviders is a composite name provider that queries all registered
// name provides in the order of registration. The first positive result
// will be returned.
type nameProviders struct {
	lock     sync.Mutex
	children []NameProvider
}

func (a *nameProviders) GetSubsystemFor(queueName string) (subsystem string, ok bool) {
	for _, child := range a.children {
		subsystem, ok = child.GetSubsystemFor(queueName)
		if ok {
			return
		}
	}
	return "", false
}

func (a *nameProviders) MustGetSubsystemFor(queueName string) (subsystem string) {
	subsystem, ok := a.GetSubsystemFor(queueName)
	if !ok {
		panic(fmt.Sprintf(
			"failed to map workqueue name '%s' to subsystem name: "+
				"none of the registered name providers provided a subsystem name",
			queueName,
		))
	}
	return
}

func (a *nameProviders) register(child NameProvider) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.children = append(a.children, child)
}

// RegisterNameProvider registers a new NameProvider.
func RegisterNameProvider(provider NameProvider) {
	nameProvidersInstance.register(provider)
}
