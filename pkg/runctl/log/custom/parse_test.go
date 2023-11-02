package custom

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_GetLoggingDetailsProvider_invalidYaml(t *testing.T) {
	t.Parallel()

	// SETUP
	g := NewGomegaWithT(t)

	const invalidYaml = "invalid1"

	// EXERCISE
	_, err := GetLoggingDetailsProvider(invalidYaml)

	// VERIFY
	g.Expect(err).To(HaveOccurred())
}

func Test_GetLoggingDetailsProvider_success(t *testing.T) {
	t.Parallel()

	// SETUP
	g := NewGomegaWithT(t)

	config := fixIndent(`
		- logKey: logKey1
		  kind: label
		  spec:
		    key: label1
		- logKey: logKey2
		  kind: annotation
		  spec:
		    key: annotation1
	`)

	run := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"annotation1": "annotation1value",
			},
			Labels: map[string]string{
				"label1": "label1value",
			},
		},
	}

	// EXERCISE
	result, err := GetLoggingDetailsProvider(config)

	// VERIFY
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).NotTo(BeNil())

	logDetails := result(run)
	g.Expect(logDetails).To(HaveExactElements(
		"logKey1", "label1value",
		"logKey2", "annotation1value",
	))
}

func Test_GetLoggingDetailsProvider_emptyConfig(t *testing.T) {
	t.Parallel()

	// SETUP
	g := NewGomegaWithT(t)

	config := ""

	// EXERCISE
	result, err := GetLoggingDetailsProvider(config)

	// VERIFY
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func Test_GetLoggingDetailsProvider_unsupported(t *testing.T) {
	t.Parallel()

	// SETUP
	g := NewGomegaWithT(t)

	config := fixIndent(`
		- logKey: logKey1
		  kind: unsupported1
		  spec: {}
		- logKey: logKey2
		  kind: unsupported2
		  spec: {}
	`)

	// EXERCISE
	result, err := GetLoggingDetailsProvider(config)

	// VERIFY
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func Test_GetLoggingDetailsProvider_ConfigError(t *testing.T) {
	t.Parallel()

	// SETUP
	g := NewGomegaWithT(t)

	config := fixIndent(`
		- logKey: logKey1
		  kind: label
		  spec:
		    # key is missing here
	`)

	// EXERCISE
	result, err := GetLoggingDetailsProvider(config)

	// VERIFY
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}
