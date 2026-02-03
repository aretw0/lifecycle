package runtime

import (
	"reflect"
	"runtime"
)

// GetFunctionName returns the name of the function or method.
// It uses reflection to retrieve the runtime name.
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
