package util

import (
	"runtime"
	"testing"
)

func TestCurrentMethod_CoverAll(t *testing.T) {
	// --- обычный путь ---
	name := CurrentMethod(1)
	if name == "unknown" {
		t.Errorf("Expected a function name, got 'unknown'")
	}

	// --- fn == nil ---
	origFuncForPC := funcForPC
	funcForPC = func(pc uintptr) *runtime.Func { return nil }
	defer func() { funcForPC = origFuncForPC }()

	name = CurrentMethod(0)
	if name != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", name)
	}

	// --- ok == false ---
	origCaller := caller
	caller = func(skip int) (uintptr, string, int, bool) { return 0, "", 0, false }
	defer func() { caller = origCaller }()

	name = CurrentMethod(0)
	if name != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", name)
	}
}
