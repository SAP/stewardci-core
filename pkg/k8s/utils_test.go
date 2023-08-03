package k8s

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type sampleTypeImplementingTypeInfoAccessOnValue struct {
	A string
	B string
}

func (r sampleTypeImplementingTypeInfoAccessOnValue) GetAPIVersion() string {
	return r.A
}

func (r sampleTypeImplementingTypeInfoAccessOnValue) GetKind() string {
	return r.B
}

func Test_TryExtractTypeInfo(t *testing.T) {
	t.Parallel()

	sampleTypeMeta := metav1.TypeMeta{APIVersion: "apiVersion1", Kind: "kind1"}

	for _, tc := range []struct {
		Name     string
		Obj      interface{}
		Expected *metav1.TypeMeta
	}{
		{
			Name:     "obj is nil",
			Obj:      nil,
			Expected: nil,
		},
		{
			Name:     "obj has no type info",
			Obj:      0,
			Expected: nil,
		},
		{
			Name:     "obj is TypeMeta",
			Obj:      sampleTypeMeta,
			Expected: &sampleTypeMeta,
		},
		{
			Name:     "obj is *TypeMeta",
			Obj:      &sampleTypeMeta,
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj has type T and *T implements access methods",
			Obj: metav1unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apiVersion1",
					"kind":       "kind1",
				},
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is pointer that has access methods",
			Obj: &metav1unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apiVersion1",
					"kind":       "kind1",
				},
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is non-pointer that has access methods",
			Obj: sampleTypeImplementingTypeInfoAccessOnValue{
				A: "apiVersion1",
				B: "kind1",
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "object is pointer of type that has access methods",
			Obj: &sampleTypeImplementingTypeInfoAccessOnValue{
				A: "apiVersion1",
				B: "kind1",
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is struct with embedded TypeMeta",
			Obj: struct {
				metav1.TypeMeta
			}{
				TypeMeta: sampleTypeMeta,
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is pointer to struct with embedded TypeMeta",
			Obj: &struct {
				metav1.TypeMeta
			}{
				TypeMeta: sampleTypeMeta,
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is struct with field TypeMeta of type TypeMeta",
			Obj: struct {
				TypeMeta metav1.TypeMeta
			}{
				TypeMeta: sampleTypeMeta,
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is pointer to struct with field TypeMeta of type TypeMeta",
			Obj: &struct {
				TypeMeta metav1.TypeMeta
			}{
				TypeMeta: sampleTypeMeta,
			},
			Expected: &sampleTypeMeta,
		},
		{
			Name: "obj is struct with field TypeMeta of type *TypeMeta",
			Obj: struct {
				*metav1.TypeMeta
			}{
				TypeMeta: &sampleTypeMeta,
			},
			Expected: nil, // should not be recognized
		},
		{
			Name: "obj is struct with field TypeMeta of other type",
			Obj: struct {
				TypeMeta struct{}
			}{},
			Expected: nil,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			tc := tc
			//t.Parallel()

			// SETUP
			g := NewGomegaWithT(t)

			// EXERCISE
			result := TryExtractTypeInfo(tc.Obj)

			// VERIFY
			if tc.Expected == nil {
				g.Expect(result).To(BeNil())
			} else {
				g.Expect(result).To(Equal(tc.Expected))
			}
		})
	}
}
