# Contextual and structured logging

Stewardci-core project supports contextual and structured logging.

[Structured logging][kep_structured_logging] generates easy to comprehend log
messages and standardizes them using a uniform format.

```text
    <log-message> <key1>=<value1> <key2>=<value2>
```

[Contextual logging][kep_contextual_logging] removes the restriction of sticking
with `k8s.io/klog` package for logging API calls.
Instead the project uses `logr.Logger` as an abstraction for logging API calls.
Contextual logging enables the caller of a function to pass a logger to function
as `logr.Logger` or `context.Context` object.
This logger may contain contextual parameters for logging as key-value pairs.

An embedded logger in `context.Context` object can be retrieved -

```go
    logr.Logger.FromContext(ctx)
```

Additionally, the log messages adhere to [the best practices and guidelines][k8s_community_logging]
advocated by Kubernetes community.

Follow Kubernetes community's guidelines and best practices for
[structured and contextual logging migration][k8s_community_structured_logging_migration_guide]
while contributing to this project.

## Limitations

klog (`k8s.io/klog`) is used as an underlying logger implementation for writing
the log messages.
This implementation does not de-duplicate key-value pairs added to logger's
context.
Therefore, the logger writes duplicate key-value pairs after a log message for
some cases.

For example:

```go
    logger := klog.Background()
    logger = logger.WithValues("foo", "abc")
    logger = logger.WithValues("foo", "abc")

    logger.Info("hello world")
```

Log entries:

```text
    I0718 00:17:47.023275   33834 main.go:27] "hello world" foo="abc" foo="abc"
```

The limitation is described in [klog #377][klog_issue_377] and states that
de-duplication is avoided in klog package for performance optimization.

Duplicate key-value pairs in log entries could occur during logging inside run
controller's reconciliation function.

[kep_structured_logging]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/1602-structured-logging
[kep_contextual_logging]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/3077-contextual-logging
[k8s_community_logging]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
[klog_issue_377]: https://github.com/kubernetes/klog/issues/377
[k8s_community_structured_logging_migration_guide]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/migration-to-structured-logging.md
