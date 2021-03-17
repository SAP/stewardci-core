package stewardlabels

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
propagate propagates specified labels from `sourceObj` to `destObj`,
while values can be enforced for individual label keys.
In case of a value conflict an error is returned and `destObj` is
not modified.

The set of labels to propagate is defined by the keys of `labelSpec`.
A label value is enforced if the corresponding key in `labelSpec` is
associated with a non-empty string value (empty values cannot be
enforced).

A value conflict exists for a label key if
	a) the label value is enforced and the existing value at `sourceObj`
	   differs from the enforced value, or
	b) the label value is enforced and the exiting value at `destObj`
	   differs from the enforced value, or
    c) the existing values at `sourceObj` and `destObject` differ.
The empty string value is NOT treated specially, e.g. there's a conflict
if `destObj` has a label set with value "foo" but `sourceObj` has the
same label set with an empty string value.

If a value is enforced for a key, the label gets set on `destObj` even
if `sourceObj` does not have that label set.
If a value is not enforced and `sourceObj` has that label set, `destObj`
get the same label set, except there is a value confict which leads to an
error.
If a value is not enforced and `sourceObj` does NOT have that label set,
the respective label at `destObj` remains unchanged, i.e. will not be
created, deleted or modified.
*/
func propagate(destObj metav1.Object, sourceObj metav1.Object, labelSpec map[string]string) error {
	sourceLabels := sourceObj.GetLabels()

	// fail if source has any value conflict with enforced value
	for k, v := range labelSpec {
		if v != "" { // value enforced
			sourceValue, found := sourceLabels[k]
			if found && sourceValue != v {
				return fmt.Errorf(
					"value conflict: source object label %q has value %q but %q is expected",
					k, sourceValue, v,
				)
			}
		}
	}

	destLabels := destObj.GetLabels()

	// don't modify labels of dest object until propagation finished without errors
	propagatedLabels := make(map[string]string)

	// propagate
	for k, v := range labelSpec {
		if v == "" { // value not enforced
			var foundOnSource bool
			v, foundOnSource = sourceLabels[k]
			if !foundOnSource {
				// do not touch this label on dest object
				continue
			}
		}
		destValue, found := destLabels[k]
		if found {
			if destValue != v {
				return fmt.Errorf(
					"value conflict: destination object label %q has existing value %q but %q is expected",
					k, destValue, v,
				)
			}
		} else {
			propagatedLabels[k] = v
		}
	}

	// don't modify if nothing is propagated
	if len(propagatedLabels) > 0 {
		if destLabels == nil {
			destObj.SetLabels(propagatedLabels)
		} else {
			for k, v := range propagatedLabels {
				// Write into the same map we originally got from dest object
				// so that we do not replace it in case it is dest object's
				// internal label map
				destLabels[k] = v
			}
			// because destLabels MAY NOT be dest object's internal label map
			// but a copy or built from some other internal representation,
			// we must call SetLabels() to ensure the update happens.
			destObj.SetLabels(destLabels)
		}
	}
	return nil
}
