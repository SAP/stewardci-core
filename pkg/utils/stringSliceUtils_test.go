package utils

import (
	"strconv"
	"testing"

	"gotest.tools/assert"
)

func Test_AddStringIfMissing_ToEmptyList(t *testing.T) {
	old := []string{}
	changed, new := AddStringIfMissing(old, "a")
	assert.DeepEqual(t, []string{"a"}, new)
	assert.Assert(t, changed)
}

func Test_AddStringIfMissing_NewElement(t *testing.T) {
	old := []string{"a"}
	changed, new := AddStringIfMissing(old, "b")
	assert.DeepEqual(t, []string{"a", "b"}, new)
	assert.Assert(t, changed)
}

func Test_AddStringIfMissing_ExistingElement(t *testing.T) {
	old := []string{"a"}
	changed, new := AddStringIfMissing(old, "a")
	assert.DeepEqual(t, []string{"a"}, new)
	assert.Assert(t, !changed)
}

func Test_RemoveString_LastAndOnlyElement(t *testing.T) {
	old := []string{"a"}
	changed, new := RemoveString(old, "a")
	assert.DeepEqual(t, []string{}, new)
	assert.Assert(t, changed)
}

func Test_RemoveString_FirstElement(t *testing.T) {
	old := []string{"a", "b"}
	changed, new := RemoveString(old, "a")
	assert.DeepEqual(t, []string{"b"}, new)
	assert.Assert(t, changed)
}

func Test_RemoveString_LastElement(t *testing.T) {
	old := []string{"a", "b"}
	changed, new := RemoveString(old, "b")
	assert.DeepEqual(t, []string{"a"}, new)
	assert.Assert(t, changed)
}

func Test_RemoveString_NoExistingElement(t *testing.T) {
	old := []string{"a", "b"}
	changed, new := RemoveString(old, "c")
	assert.DeepEqual(t, []string{"a", "b"}, new)
	assert.Assert(t, !changed)
}

func Test_StringSliceContains(t *testing.T) {
	for i, tc := range []struct {
		slice    []string
		elem     string
		expected bool
	}{
		{[]string{}, "", false},
		{[]string{}, "x", false},
		{[]string{""}, "", true},
		{[]string{"x"}, "", false},
		{[]string{"x"}, "a", false},
		{[]string{"x"}, "x", true},
		{[]string{"x", "y", "z"}, "a", false},
		{[]string{"x", "y", "z"}, "x", true},
		{[]string{"x", "y", "z"}, "y", true},
		{[]string{"x", "y", "z"}, "z", true},
	} {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result := StringSliceContains(tc.slice, tc.elem)
			assert.Equal(t, tc.expected, result)
		})
	}
}
