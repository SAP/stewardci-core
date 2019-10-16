package v1alpha1_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

func Test_CustomJSON_Marshal(t *testing.T) {
	// SETUP
	value := map[string]interface{}{
		"foo": "bar",
	}
	examinee := v1alpha1.CustomJSON{value}

	// EXERCISE
	data, err := json.Marshal(&examinee)
	assert.NilError(t, err)

	// VERIFY
	encoded := string(data)
	assert.Equal(t, `{"foo":"bar"}`, encoded)
}

func Test_CustomJSON_Marshal_NilPtr(t *testing.T) {
	// SETUP
	nilPtr := (*v1alpha1.CustomJSON)(nil)

	// EXERCISE
	data, err := json.Marshal(&nilPtr)
	assert.NilError(t, err)

	// VERIFY
	encoded := string(data)
	assert.Equal(t, `null`, encoded)
}

func Test_CustomJSON_MarshalJSON_WithNilReceiver(t *testing.T) {
	// SETUP
	examineeFunc := (*v1alpha1.CustomJSON).MarshalJSON

	// EXERCISE
	data, err := examineeFunc(nil)
	assert.NilError(t, err)

	// VERIFY
	encoded := string(data)
	assert.Equal(t, `null`, encoded)
}

func Test_CustomJSON_Unmarshal(t *testing.T) {
	// SETUP
	encoded := []byte(`{"foo":"bar"}`)
	examinee := v1alpha1.CustomJSON{}

	// EXERCISE
	err := json.Unmarshal(encoded, &examinee)
	assert.NilError(t, err)

	// VERIFY
	expectedValue := map[string]interface{}{
		"foo": "bar",
	}
	assert.DeepEqual(t, expectedValue, examinee.Value)
}

func Test_CustomJSON_UnmarshalJSON_WithWrongInput(t *testing.T) {
	// SETUP
	encoded := []byte(`["x", abc]`)
	examinee := v1alpha1.CustomJSON{"previous"}

	// EXERCISE
	// Cannot test via json.Unmarshal because it will catch
	// and report the erroneous JSON input before delegating
	// to CustomJSON.UnmarshalJSON
	err := examinee.UnmarshalJSON(encoded)

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "previous", examinee.Value)
}

func Test_CustomJSON_UnmarshalJSON_PanicsOnNilReceiver(t *testing.T) {
	// SETUP
	encoded := []byte(`{"foo":"bar"}`)
	examineeFunc := (*v1alpha1.CustomJSON).UnmarshalJSON

	// EXERCISE
	assert.Assert(t, cmp.Panics(func() {
		examineeFunc(nil, encoded)
	}))
}

func Test_CustomJSON_DeepCopy_WithSerializableValues(t *testing.T) {
	testparams := []struct {
		in  interface{}
		out interface{}
	}{
		{nil, nil},
		{true, true},
		{false, false},
		{1.5, 1.5},
		{
			// copy has differnt type
			in:  1,
			out: 1.0,
		},
		{"foo", "foo"},
		{
			in:  map[string]interface{}{"foo": "bar"},
			out: map[string]interface{}{"foo": "bar"},
		},
		{
			// copy has different (generic) type
			in:  map[string]string{"foo": "bar"},
			out: map[string]interface{}{"foo": "bar"},
		},
		{
			in:  []interface{}{"foo", "bar"},
			out: []interface{}{"foo", "bar"},
		},
		{
			// copy has different (generic) type
			in:  []string{"foo", "bar"},
			out: []interface{}{"foo", "bar"},
		},
	}

	/**
	 * Test
	 */
	test := "DeepCopyInto"
	for i, p := range testparams {
		t.Run(test+"_"+strconv.Itoa(i), func(t *testing.T) {
			// SETUP
			examinee := v1alpha1.CustomJSON{p.in}
			copy := v1alpha1.CustomJSON{}

			// EXERCISE
			examinee.DeepCopyInto(&copy)

			// VERIFY
			assert.DeepEqual(t, p.out, copy.Value)
		})
	}

	/**
	 * Test
	 */
	test = "DeepCopy"
	for i, p := range testparams {
		t.Run(test+"_"+strconv.Itoa(i), func(t *testing.T) {
			// SETUP
			examinee := v1alpha1.CustomJSON{p.in}

			// EXERCISE
			copy := examinee.DeepCopy()

			// VERIFY
			assert.DeepEqual(t, p.out, copy.Value)
		})
	}
}

func Test_CustomJSON_DeepCopy_WithUnserializableValue(t *testing.T) {
	unserializableValue := map[string]interface{}{
		"foo": []interface{}{
			"bar",
			1.0i, // complex not serializable
		},
	}

	/**
	 * Test
	 */
	t.Run("DeepCopyInto", func(t *testing.T) {
		// SETUP
		examinee := v1alpha1.CustomJSON{unserializableValue}
		copy := v1alpha1.CustomJSON{}

		// EXERCISE
		assert.Assert(t, cmp.Panics(func() {
			examinee.DeepCopyInto(&copy)
		}))
	})

	/**
	 * Test
	 */
	t.Run("DeepCopy", func(t *testing.T) {
		// SETUP
		examinee := v1alpha1.CustomJSON{unserializableValue}

		// EXERCISE
		assert.Assert(t, cmp.Panics(func() {
			examinee.DeepCopy()
		}))
	})
}

func Test_CustomJSON_DeepCopyInto_PanicsOnNilReceiver(t *testing.T) {
	examineeFunc := (*v1alpha1.CustomJSON).DeepCopyInto

	assert.Assert(t, cmp.Panics(func() {
		examineeFunc(nil, &v1alpha1.CustomJSON{})
	}))
}

func Test_CustomJSON_DeepCopy_WithNilReceiver(t *testing.T) {
	examineeFunc := (*v1alpha1.CustomJSON).DeepCopy

	result := examineeFunc(nil)

	assert.Equal(t, (*v1alpha1.CustomJSON)(nil), result)
}
