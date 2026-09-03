package impact

import (
	"reflect"
	"testing"
)

func TestScanESModuleDependencies(t *testing.T) {
	src := []byte(`
import React from "react";
import "./reset.css";
export { Button } from './Button';
const lazy = import('./LazyPanel');
const unknown = import(resolvePanel());
// import "./ignored-comment";
/* export * from "./ignored-block"; */
`)
	got := ScanESModuleDependencies(src)
	if !reflect.DeepEqual(got.StaticSpecifiers, []string{"./Button", "./reset.css", "react"}) {
		t.Fatalf("static specifiers = %#v", got.StaticSpecifiers)
	}
	if !reflect.DeepEqual(got.DynamicSpecifiers, []string{"./LazyPanel"}) {
		t.Fatalf("dynamic specifiers = %#v", got.DynamicSpecifiers)
	}
	if !got.DynamicUnresolved {
		t.Fatal("non-literal dynamic import must be surfaced as unresolved")
	}
}

func TestScanESModuleDependenciesLiteralDynamicIsResolved(t *testing.T) {
	got := ScanESModuleDependencies([]byte(`const view = import("./View");`))
	if got.DynamicUnresolved {
		t.Fatal("literal dynamic import should not be marked unresolved")
	}
	if !reflect.DeepEqual(got.DynamicSpecifiers, []string{"./View"}) {
		t.Fatalf("dynamic specifiers = %#v", got.DynamicSpecifiers)
	}
}
