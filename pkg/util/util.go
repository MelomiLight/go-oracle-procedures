package util

import (
	"runtime"
	"strings"
)

var caller = runtime.Caller
var funcForPC = runtime.FuncForPC

/*
CurrentMethod returns the name of the current method.
For example, service.(*ProcedureService).CallProcedure
skip is the number of stack frames to skip.
*/
func CurrentMethod(skip int) string {
	pc, _, _, ok := caller(skip)
	if !ok {
		return "unknown"
	}
	fn := funcForPC(pc)
	if fn == nil {
		return "unknown"
	}

	parts := strings.Split(fn.Name(), "/")
	return parts[len(parts)-1]
}
