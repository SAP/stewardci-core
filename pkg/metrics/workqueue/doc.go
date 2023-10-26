/*
Package workqueue embeds metrics exposed by package
k8s.io/client-go/util/workqueue.

Packages that make use of workqueue, e.g. runctl, must contribute a
NameProvider that maps names of workqueues to subsystem names (i.e. metric name
prefixes), so that workqueue metrics appear under the same prefix as other
metrics from that package.
*/
package workqueue
