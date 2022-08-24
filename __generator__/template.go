package main

const predefinedVariables = `
// Code generated by __generator__/predefined_variables.go; DO NOT EDIT.

package context

import (
	"github.com/Financial-Times/falco/types"
)

func predefinedVariables() Variables {
	return {{ .Variables }}
}

func newRegexMatchedValues() map[string]int {
	return map[string]int{
		"re.group.0": 0,
		"re.group.1": 0,
		"re.group.2": 0,
		"re.group.3": 0,
		"re.group.4": 0,
		"re.group.5": 0,
		"re.group.6": 0,
		"re.group.7": 0,
		"re.group.8": 0,
		"re.group.9": 0,
		"re.group.10": 0,
	}
}`

const builtinFunctions = `
// Code generated by __generator__/builtin_functions.go; DO NOT EDIT.

package context

import (
	"github.com/Financial-Times/falco/types"
)

type Functions map[string]*FunctionSpec

type FunctionSpec struct {
	Items map[string]*FunctionSpec
	Value *BuiltinFunction
}

type BuiltinFunction struct {
	Arguments 						[][]types.Type
	Return    						types.Type
	Extra     						func(c *Context, name string) interface{}
	Scopes    						int
	Reference 						string
	IsUserDefinedFunction bool
}

func builtinFunctions() Functions {
	return {{ .Functions }}
}`
