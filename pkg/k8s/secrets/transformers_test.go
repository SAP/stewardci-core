package secrets

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_UniqueNameTransformer_WithNameSet(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretWithType("name1", "secret1", v1.SecretTypeDockercfg)
	transformed := orig.DeepCopy()

	// EXERCISE
	UniqueNameTransformer()(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetName("")
	expected.SetGenerateName("name1-")

	assert.DeepEqual(t, expected, transformed)
}

func Test_RenameByAnnotationTransformer(t *testing.T) {
	t.Parallel()
	const originalName string = "orig1"
	for _, tc := range []struct {
		name         string
		annotations  map[string]string
		key          string
		expectedName string
	}{
		{
			name:         "no_annotation",
			annotations:  map[string]string{},
			key:          "any",
			expectedName: originalName,
		}, {
			name:         "working_rename",
			annotations:  map[string]string{"key1": "newName1"},
			key:          "key1",
			expectedName: "newName1",
		}, {
			name:         "transformer_with_wrong_key",
			annotations:  map[string]string{"key1": "newName1"},
			key:          "wrong_key",
			expectedName: originalName,
		}, {
			name:         "transformer_with_empty_key",
			annotations:  map[string]string{"key1": "newName1"},
			key:          "",
			expectedName: originalName,
		}, {
			name:         "annotation_has_empty_new_name",
			annotations:  map[string]string{"key1": ""},
			key:          "key1",
			expectedName: originalName,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// SETUP

			orig := fake.SecretWithType(originalName, "secret1", v1.SecretTypeDockercfg)
			orig.SetAnnotations(tc.annotations)
			transformed := orig.DeepCopy()

			// EXERCISE
			RenameByAnnotationTransformer(tc.key)(transformed)

			// VERIFY
			expected := orig.DeepCopy()
			expected.SetName(tc.expectedName)
			assert.DeepEqual(t, expected, transformed)
		})
	}
}

func Test_SetAnnotationTransformer_SetNew(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no annotations
	transformed := orig.DeepCopy()

	// EXERCISE
	SetAnnotationTransformer("foo", "bar")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{
		"foo": "bar",
	})

	assert.DeepEqual(t, expected, transformed)
}

func Test_SetAnnotationTransformer_OverwriteExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetAnnotations(map[string]string{
		"foo": "origValue1",
	})
	transformed := orig.DeepCopy()

	// EXERCISE
	SetAnnotationTransformer("foo", "newValue1")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{
		"foo": "newValue1",
	})

	assert.DeepEqual(t, expected, transformed)
}

func Test_StripAnnotationsTransformer_Match(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetAnnotations(map[string]string{
		"foo": "bar",
	})
	transformed := orig.DeepCopy()

	// EXERCISE
	StripAnnotationsTransformer("f")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{})

	assert.DeepEqual(t, expected, transformed)
}

func Test_StripAnnotationsTransformer_NoMatch(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetAnnotations(map[string]string{
		"foo": "bar",
	})
	transformed := orig.DeepCopy()

	// EXERCISE
	StripAnnotationsTransformer("x")(transformed)

	// VERIFY
	assert.DeepEqual(t, orig, transformed)
}

func Test_StripAnnotationsTransformer_NoExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no annotations
	transformed := orig.DeepCopy()

	// EXERCISE
	StripAnnotationsTransformer("f")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{})

	assert.DeepEqual(t, expected, transformed)
}

func Test_SetLabelTransformer_SetNew(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no labels
	transformed := orig.DeepCopy()

	// EXERCISE
	SetLabelTransformer("foo", "bar")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{
		"foo": "bar",
	})

	assert.DeepEqual(t, expected, transformed)
}

func Test_SetLabelTransformer_OverwriteExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetLabels(map[string]string{
		"foo": "origValue1",
	})
	transformed := orig.DeepCopy()

	// EXERCISE
	SetLabelTransformer("foo", "newValue1")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{
		"foo": "newValue1",
	})

	assert.DeepEqual(t, expected, transformed)
}

func Test_StripLabelsTransformer_Match(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetLabels(map[string]string{
		"foo": "bar",
	})
	transformed := orig.DeepCopy()

	// EXERCISE
	StripLabelsTransformer("f")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{})

	assert.DeepEqual(t, expected, transformed)
}

func Test_StripLabelsTransformer_NoMatch(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetLabels(map[string]string{
		"foo": "bar",
	})
	transformed := orig.DeepCopy()

	// EXERCISE
	StripLabelsTransformer("x")(transformed)

	// VERIFY
	assert.DeepEqual(t, orig, transformed)
}

func Test_StripLabelsTransformer_NoExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no annotations
	transformed := orig.DeepCopy()

	// EXERCISE
	StripLabelsTransformer("f")(transformed)

	// VERIFY
	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{})

	assert.DeepEqual(t, expected, transformed)
}
