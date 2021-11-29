/*

Package metrics provides metrics support shared among all packages in this Go
module:

-   the metrics registry
-   exporting metrics via HTTP
-   common metrics

It does NOT include code that is specific to other packages of this module.


Global State

The API of this packages makes use of global state to get access to instances so
that keeping and passing references is not necessary. For non-test use cases
this perfectly fits to the global nature of metric support.

For testing it may be required to let SUTs use test doubles instead of the
original global instances of this package. This can be achieved by patching the
global state of this package during test setup and reverting the patch at test
teardown. Be aware that tests patching global state must not run concurrently to
other tests to avoid interference. See the Testing type for test support.

*/
package metrics
