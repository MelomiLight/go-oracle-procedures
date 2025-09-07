package util

import (
	"runtime"
	"strings"
)

/*
CurrentMethod returns the name of the current method.
For example, service.(*ProcedureService).CallProcedure
skip is the number of stack frames to skip.
*/
func CurrentMethod(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	parts := strings.Split(fn.Name(), "/")
	return parts[len(parts)-1]
}
