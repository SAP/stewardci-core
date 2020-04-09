module github.com/SAP/stewardci-core

go 1.12

require (
	cloud.google.com/go v0.48.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/ghodss/yaml v1.0.0
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/mock v1.3.1
	github.com/google/uuid v1.1.1
	github.com/lithammer/dedent v1.1.0
	github.com/openzipkin/zipkin-go v0.2.2 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/tektoncd/pipeline v0.10.2
	go.uber.org/zap v1.13.0 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.1
	k8s.io/client-go v0.17.0
	k8s.io/kubernetes v1.11.10 // indirect
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 // indirect
	knative.dev/pkg v0.0.0-20191111150521-6d806b998379
)

replace (
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.10.2
	k8s.io/api => k8s.io/api v0.0.0-20191004102349-159aefb8556b // kubernetes-1.14.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689 // kubernetes-1.14.9
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20191029005444-8e4128053008+incompatible // kubernetes-1.14.9
	knative.dev/pkg => knative.dev/pkg v0.0.0-20191107185656-884d50f09454 // release-0.10
)
