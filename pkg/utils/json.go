package utils

import (
	"encoding/json"
	"github.com/pkg/errors"
)

// ToJSONString converts value to JSON string
func ToJSONString(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", errors.Wrapf(err, "error while serializing to JSON: %v", err)
	}
	return string(bytes), nil
}
