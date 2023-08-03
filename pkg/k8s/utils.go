package k8s

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TryExtractTypeInfo tries to extract Kubernetes API object type information
// from the given object:
//   - obj type T is "k8s.io/apimachinery/pkg/apis/meta/v1".TypeMeta or
//     *"k8s.io/apimachinery/pkg/apis/meta/v1".TypeMeta.
//   - The type T of obj or *T has the following methods:
//     GetAPIVersion() string
//     GetKind() string
//   - obj is a struct or a pointer to a struct. The struct has a field
//     named "TypeMeta" of type "k8s.io/apimachinery/pkg/apis/meta/v1".TypeMeta
//
// If type information is found, it is provides a pointer to a metav1.TypeMeta
// struct. It may point to the original object or to a newly created instance.
// Therefore, modifying the returned instance may change the original object.
//
// If obj is nil or no type information has been found, nil is returned.
func TryExtractTypeInfo(obj interface{}) *metav1.TypeMeta {
	type Typed interface {
		GetAPIVersion() string
		GetKind() string
	}

	switch v := obj.(type) {
	case nil:
		return nil
	case *metav1.TypeMeta:
		return v
	case metav1.TypeMeta:
		return &v
	case Typed:
		return &metav1.TypeMeta{
			APIVersion: v.GetAPIVersion(),
			Kind:       v.GetKind(),
		}
	}

	objValue := reflect.ValueOf(obj)

	// *T implements Typed
	{
		pointerValue := reflect.New(objValue.Type())
		pointerValue.Elem().Set(objValue)
		if v, ok := pointerValue.Interface().(Typed); ok {
			return &metav1.TypeMeta{
				APIVersion: v.GetAPIVersion(),
				Kind:       v.GetKind(),
			}
		}
	}

	// If T is pointer, continue further check on value that
	// it points to
	if objValue.Kind() == reflect.Pointer {
		objValue = objValue.Elem()
	}

	// T is struct with field TypeMeta of type metav1.TypeMeta
	if objValue.Kind() == reflect.Struct {
		typeMetaValue := objValue.FieldByName("TypeMeta")
		if typeMetaValue.IsValid() && typeMetaValue.Type() == reflect.TypeOf(metav1.TypeMeta{}) {
			typeMeta := typeMetaValue.Interface().(metav1.TypeMeta)
			return &typeMeta
		}
	}

	return nil
}
