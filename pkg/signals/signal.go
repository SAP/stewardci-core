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

	"github.com/go-logr/logr"
)

var onlyOneShutdownSignalHandler = make(chan struct{})
var onlyOneThreaddumpSignalHandler = make(chan struct{})

// SetupShutdownSignalHandler registeres a handler for SIGTERM and SIGINT. A
// stop channel is returned which is closed on the first receipt of one of these
// signals. On a second receipt the provided killFunc is called in a new
// gorouting. Upon a third receipt the process gets terminated via os.Exit(1).
//
// killFunc is supposed to shutdown the process without further significant delay.
func SetupShutdownSignalHandler(logger logr.Logger, killFunc func()) (stopCh <-chan struct{}) {
	close(onlyOneShutdownSignalHandler) // panics when called twice
	stop := make(chan struct{})
	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, shutdownSignals...)
	go func() {
		sig := <-sigs
		logSignalReceived(logger, sig)
		logger.Info("Initiating graceful shutdown")
		close(stop)

		sig = <-sigs
		logSignalReceived(logger, sig)
		logger.Info("Invoking kill function after second shutdown signal")
		go killFunc()

		sig = <-sigs
		logSignalReceived(logger, sig)
		logger.Info("Exiting immediately after third shutdown signal")
		os.Exit(1)
	}()
	return stop
}

// SetupThreadDumpSignalHandler registers a handler for SIGQUIT. Each time the
// signal is received, a thread dump is logged.
func SetupThreadDumpSignalHandler(logger logr.Logger) {
	close(onlyOneThreaddumpSignalHandler) // panics when called twice
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		buf := make([]byte, 1*1024*1024)
		for {
			sig := <-sigs
			logSignalReceived(logger, sig)
			stacklen := runtime.Stack(buf, true)
			logger.Info("Goroutine dump", "dump", buf[:stacklen])
		}
	}()
}

func logSignalReceived(logger logr.Logger, sig os.Signal) {
	logger.Info("Received signal", "signal", sig)
}
