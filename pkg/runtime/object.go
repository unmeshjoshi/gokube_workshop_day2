package runtime

import (
	"encoding/json"
	"fmt"
)

// Object is a marker interface for Kubernetes-like API objects
type Object interface{}

// Encode serializes an Object to JSON
func Encode(obj Object) ([]byte, error) {
	return json.Marshal(obj)
}

// Decode deserializes JSON data into an Object
func Decode(data []byte, obj Object) error {
	return json.Unmarshal(data, obj)
}

// GetObjectKind returns the kind of the object
func GetObjectKind(obj Object) string {
	return fmt.Sprintf("%T", obj)
}
