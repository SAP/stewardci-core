/*
Source: https://github.com/kubernetes/sample-controller/tree/master/pkg/signals

Copyright 2017 The Kubernetes Authors.

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

package signals

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	klog "k8s.io/klog/v2"
)

var onlyOneShutdownSignalHandler = make(chan struct{})
var onlyOneThreaddumpSignalHandler = make(chan struct{})

// SetupShutdownSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupShutdownSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneShutdownSignalHandler) // panics when called twice
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()
	return stop
}

// SetupThreadDumpSignalHandler registers a handler for SIGQUIT.
// In case a SIGQUIT is received a thread dump is written.
func SetupThreadDumpSignalHandler() {
	close(onlyOneThreaddumpSignalHandler) // panics when called twice
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		buf := make([]byte, 1*1024*1024)
		for {
			sig := <-sigs
			stacklen := runtime.Stack(buf, true)
			klog.InfoS("Received signal", "signal", sig)
			klog.InfoS("Goroutine dump", "dump", buf[:stacklen])
		}
	}()
}
