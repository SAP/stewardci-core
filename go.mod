module github.com/SAP/stewardci-core

go 1.14

require (
	cloud.google.com/go v0.58.0 // indirect
	github.com/Azure/azure-sdk-for-go v43.2.0+incompatible // indirect
	github.com/aws/aws-sdk-go v1.32.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.4.3
	github.com/google/go-containerregistry v0.1.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/lithammer/dedent v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.10.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/tektoncd/pipeline v0.13.2
	github.com/vdemeester/k8s-pkg-credentialprovider v1.18.0 // indirect
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9 // indirect
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	golang.org/x/text v0.3.3 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/genproto v0.0.0-20200612171551-7676ae05be11 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog/v2 v2.3.0
	k8s.io/legacy-cloud-providers v0.18.3 // indirect
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451 // indirect
	knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
)

replace (
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.8.0
	k8s.io/api => k8s.io/api v0.17.6 // kubernetes-1.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6 // kubernetes-1.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6 // kubernetes-1.17.6
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9 // release-0.15
)
