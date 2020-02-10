module github.com/SAP/stewardci-core

go 1.12

require (
	cloud.google.com/go v0.48.0 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.8 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/mock v1.3.1
	github.com/google/go-containerregistry v0.0.0-20191115225042-f8574ec722f4 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/lithammer/dedent v1.1.0
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/openzipkin/zipkin-go v0.2.2 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/tektoncd/pipeline v0.8.0
	go.uber.org/zap v1.13.0 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.0.0-20180904230853-4e7be11eab3f
	k8s.io/apimachinery v0.0.0-20180904193909-def12e63c512
	k8s.io/client-go v0.0.0-20180910083459-2cefa64ff137
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 // indirect
	knative.dev/pkg v0.0.0-00010101000000-000000000000
)

replace (
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.8.0
	k8s.io/api => k8s.io/api v0.0.0-20191004102349-159aefb8556b // kubernetes-1.14.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689 // kubernetes-1.14.9
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20191029005444-8e4128053008+incompatible // kubernetes-1.14.9
	knative.dev/pkg => knative.dev/pkg v0.0.0-20191107185656-884d50f09454 // release-0.10
)
