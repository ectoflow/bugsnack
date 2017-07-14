package hashstruct

import "reflect"

type Hash map[string]interface{}

func (hash *Hash) IsZeroInterface() bool {
	return hash == reflect.Zero(reflect.TypeOf(hash)).Interface()
}
