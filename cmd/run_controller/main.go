package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/SAP/stewardci-core/pkg/featureflag"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/SAP/stewardci-core/pkg/runctl"
	"github.com/SAP/stewardci-core/pkg/signals"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"
	"knative.dev/pkg/system"
)

const (
	// resyncPeriod is the period between full resyncs performed
	// by the controller.
	resyncPeriod = 30 * time.Second

	// metricsPort is the TCP port number to be used by the metrics
	// HTTP server.
	metricsPort = 9090
)

var (
	kubeconfig              string
	burst, qps, threadiness int

	heartbeatInterval time.Duration
	heartbeatLogging  bool
	heartbeatLogLevel int

	k8sAPIRequestTimeout time.Duration
)

func init() {
	klog.InitFlags(nil)

	flag.StringVar(
		&kubeconfig,
		"kubeconfig",
		"",
		"The path to a kubeconfig file configuring access to the Kubernetes cluster."+
			" If not specified or empty, assume running in-cluster.",
	)
	flag.IntVar(
		&qps,
		"qps",
		5,
		"The queries per seconds (QPS) for Kubernetes API client-side rate limiting.",
	)
	flag.IntVar(
		&burst,
		"burst",
		10,
		"The size of the burst bucket for Kubernetes API client-side rate limiting.",
	)
	flag.IntVar(
		&threadiness,
		"threadiness",
		2,
		"The maximum number of reconciliations performed by the controller in parallel.",
	)
	flag.DurationVar(
		&heartbeatInterval,
		"heartbeat-interval",
		1*time.Minute,
		"The interval of controller heartbeats.",
	)
	flag.BoolVar(
		&heartbeatLogging,
		"heartbeat-logging",
		true,
		"Whether controller heartbeats should be logged.",
	)
	flag.IntVar(
		&heartbeatLogLevel,
		"heartbeat-log-level",
		3,
		"The log level to be used for controller heartbeats.",
	)
	flag.DurationVar(
		&k8sAPIRequestTimeout,
		"k8s-api-request-timeout",
		15*time.Minute,
		"The maximum length of time to wait before giving up on a server request. A value of zero means no timeout.",
	)

	flag.Parse()
}

func main() {
	defer klog.Flush()

	ctx := context.Background()
	logger := klog.FromContext(ctx)

	system.Namespace() // ensure that namespace is set in environment
	featureflag.Log(logger)

	var config *rest.Config
	var err error

	if kubeconfig == "" {
		logger.Info("Loading in-cluster kube config")
		config, err = rest.InClusterConfig()
		if err != nil {
			logger.Error(err, "Failed to load kubeconfig. Hint: You can use parameter '-kubeconfig' for local testing")
			flushLogsAndExit()
		}
	} else {
		logger.Info("Loading kube config given via command line flag")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Error(err, "Failed to create kubeconfig from command line flag", "flag", "-kubeconfig", "path", kubeconfig)
			flushLogsAndExit()
		}
	}

	logger.V(3).Info("Creating client factory",
		"resyncPeriod", resyncPeriod,
		"QPS", qps,
		"burst", burst,
		"kubeAPIRequestTimeout", k8sAPIRequestTimeout,
	)

	config.QPS = float32(qps)
	config.Burst = burst
	config.Timeout = k8sAPIRequestTimeout
	factory := k8s.NewClientFactory(logger, config, resyncPeriod)

	if factory == nil {
		logger.Error(nil, "Failed to create Kubernetes clients",
			"resyncPeriod", resyncPeriod,
			"QPS", qps,
			"burst", burst,
			"kubeAPIRequestTimeout", k8sAPIRequestTimeout,
		)
		flushLogsAndExit()
	}

	logger.V(2).Info("Starting metrics server",
		"metricsEndpoint", fmt.Sprintf("http://0.0.0.0:%d/metrics", metricsPort),
	)
	metrics.StartServer(logger, metricsPort)

	logger.V(3).Info("Creating controller")
	controllerOpts := runctl.ControllerOpts{
		HeartbeatInterval:       heartbeatInterval,
		HeartbeatLoggingEnabled: heartbeatLogging,
		HeartbeatLogLevel:       heartbeatLogLevel,
	}

	controller := runctl.NewController(logger, factory, controllerOpts)

	logger.V(3).Info("Creating signal handlers")
	stopCh := signals.SetupShutdownSignalHandler(logger, flushLogsAndExit)
	signals.SetupThreadDumpSignalHandler(logger)

	logger.V(2).Info("Starting Informers")
	factory.StewardInformerFactory().Start(stopCh)
	factory.TektonInformerFactory().Start(stopCh)

	logger.V(2).Info("Running controller", "threadiness", threadiness)
	if err = controller.Run(threadiness, stopCh); err != nil {
		logger.Error(err, "Failed to run controller")
		flushLogsAndExit()
	}
}

func flushLogsAndExit() {
	klog.FlushAndExit(klog.ExitFlushTimeout, 1)
}
