module github.com/SAP/stewardci-core

go 1.14

require (
	github.com/Azure/go-autorest/autorest v0.10.2 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.3 // indirect
	github.com/aws/aws-sdk-go v1.31.12 // indirect
	github.com/containerd/containerd v1.3.4 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.4.3
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-cmp v0.4.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/lithammer/dedent v1.1.0
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/tektoncd/pipeline v0.10.2
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/grpc v1.29.1 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/legacy-cloud-providers v0.17.6 // indirect
	knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.3.0+incompatible // not used, but resolves ambiguous import from multi-module repo (see https://github.com/Azure/azure-event-hubs-go/issues/117, https://github.com/Azure/go-autorest/issues/414)
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.8.0
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
	k8s.io/api => k8s.io/api v0.17.6 // kubernetes-1.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6 // kubernetes-1.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6 // kubernetes-1.17.6
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9 // release-0.15
)
