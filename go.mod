module github.com/SAP/stewardci-core

go 1.14

require (
	cloud.google.com/go v0.58.0 // indirect
	github.com/aws/aws-sdk-go v1.34.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.4.3
	github.com/google/uuid v1.1.1
	github.com/gruntwork-io/terratest v0.27.4
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/lithammer/dedent v1.1.0
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.10.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/tektoncd/pipeline v0.14.3
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/genproto v0.0.0-20200612171551-7676ae05be11 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog/v2 v2.1.0
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451 // indirect
	knative.dev/pkg v0.0.0-20200702222342-ea4d6e985ba0
)

replace (
	k8s.io/api => k8s.io/api v0.17.13 // kubernetes-1.17.13
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.13 // kubernetes-1.17.13
	k8s.io/client-go => k8s.io/client-go v0.17.13 // kubernetes-1.17.13
)
