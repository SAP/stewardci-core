package v1alpha1

import "encoding/json"

// CustomJSON is used for fields where any JSON value is allowed.
// It exists only to provide deep copy methods.
// The zero value represents a JSON null value.
type CustomJSON struct {
	Value interface{}
}

// ensure that CustomJSON implements the required interfaces
var _ json.Marshaler = (*CustomJSON)(nil)
var _ json.Unmarshaler = (*CustomJSON)(nil)

// MarshalJSON fulfills interface encoding.json.Marshaler
func (c *CustomJSON) MarshalJSON() ([]byte, error) {
	var v *interface{}
	if c != nil {
		v = &c.Value
	}
	return json.Marshal(v)
}

// UnmarshalJSON fulfills interface encoding.json.Unmarshaler
func (c *CustomJSON) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	*c = CustomJSON{value}
	return nil
}

// DeepCopyInto writes a deep copy of the receiver into out. c must be non-nil.
func (c *CustomJSON) DeepCopyInto(out *CustomJSON) {
	_ = c.Value // panic if c == nil
	bytes, err := c.MarshalJSON()
	if err != nil {
		panic(err)
	}
	err = out.UnmarshalJSON(bytes)
	if err != nil {
		panic(err)
	}
}

// DeepCopy creates a new CustomJSON as a deep copy of the receiver.
func (c *CustomJSON) DeepCopy() *CustomJSON {
	if c == nil {
		return nil
	}
	copy := new(CustomJSON)
	c.DeepCopyInto(copy)
	return copy
}
