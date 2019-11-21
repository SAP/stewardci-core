package secrets

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_UniqueNameTransformer_WithNameSet(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretWithType("name1", "secret1", v1.SecretTypeDockercfg)
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := UniqueNameTransformer()(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetName("")
	expected.SetGenerateName("name1-")

	assert.DeepEqual(t, expected, result)
}

func Test_SetAnnotationTransformer_SetNew(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no annotations
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := SetAnnotationTransformer("foo", "bar")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{
		"foo": "bar",
	})

	assert.DeepEqual(t, expected, result)
}

func Test_SetAnnotationTransformer_OverwriteExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetAnnotations(map[string]string{
		"foo": "origValue1",
	})
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := SetAnnotationTransformer("foo", "newValue1")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{
		"foo": "newValue1",
	})

	assert.DeepEqual(t, expected, result)
}

func Test_StripAnnotationsTransformer_Match(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetAnnotations(map[string]string{
		"foo": "bar",
	})
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := StripAnnotationsTransformer("f")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{})

	assert.DeepEqual(t, expected, result)
}

func Test_StripAnnotationsTransformer_NoMatch(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetAnnotations(map[string]string{
		"foo": "bar",
	})
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := StripAnnotationsTransformer("x")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	assert.DeepEqual(t, orig, result)
}

func Test_StripAnnotationsTransformer_NoExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no annotations
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := StripAnnotationsTransformer("f")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetAnnotations(map[string]string{})

	assert.DeepEqual(t, expected, result)
}

func Test_SetLabelTransformer_SetNew(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no labels
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := SetLabelTransformer("foo", "bar")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{
		"foo": "bar",
	})

	assert.DeepEqual(t, expected, result)
}

func Test_SetLabelTransformer_OverwriteExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetLabels(map[string]string{
		"foo": "origValue1",
	})
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := SetLabelTransformer("foo", "newValue1")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{
		"foo": "newValue1",
	})

	assert.DeepEqual(t, expected, result)
}

func Test_StripLabelsTransformer_Match(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetLabels(map[string]string{
		"foo": "bar",
	})
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := StripLabelsTransformer("f")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{})

	assert.DeepEqual(t, expected, result)
}

func Test_StripLabelsTransformer_NoMatch(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1")
	orig.SetLabels(map[string]string{
		"foo": "bar",
	})
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := StripLabelsTransformer("x")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	assert.DeepEqual(t, orig, result)
}

func Test_StripLabelsTransformer_NoExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	orig := fake.SecretOpaque("name1", "secret1") // no annotations
	origCopy := orig.DeepCopy()

	// EXERCISE
	result := StripLabelsTransformer("f")(origCopy)

	// VERIFY
	assert.DeepEqual(t, orig, origCopy)
	assert.Assert(t, result != origCopy)

	expected := orig.DeepCopy()
	expected.SetLabels(map[string]string{})

	assert.DeepEqual(t, expected, result)
}
