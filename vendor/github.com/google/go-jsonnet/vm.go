/*
Copyright 2017 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/internal/program"
)

// Note: There are no garbage collection params because we're using the native
// Go garbage collector.

// VM is the core interpreter and is the touchpoint used to parse and execute
// Jsonnet.
type VM struct {
	MaxStack       int
	ext            vmExtMap
	tla            vmExtMap
	nativeFuncs    map[string]*NativeFunction
	importer       Importer
	ErrorFormatter ErrorFormatter
	StringOutput   bool
	importCache    *importCache
}

// External variable or top level argument provided before execution
type vmExt struct {
	// jsonnet code to evaluate or string to pass
	value string
	// isCode determines whether it should be evaluated as jsonnet code or
	// treated as string.
	isCode bool
}

type vmExtMap map[string]vmExt

// MakeVM creates a new VM with default parameters.
func MakeVM() *VM {
	defaultImporter := &FileImporter{}
	return &VM{
		MaxStack:       500,
		ext:            make(vmExtMap),
		tla:            make(vmExtMap),
		nativeFuncs:    make(map[string]*NativeFunction),
		ErrorFormatter: &termErrorFormatter{pretty: false, maxStackTraceSize: 20},
		importer:       &FileImporter{},
		importCache:    makeImportCache(defaultImporter),
	}
}

// Fully flush cache. This should be executed when we are no longer sure that the source files
// didn't change, for example when the importer changed.
func (vm *VM) flushCache() {
	vm.importCache = makeImportCache(vm.importer)
}

// Flush value cache. This should be executed when calculated values may no longer be up to date,
// for example due to change in extVars.
func (vm *VM) flushValueCache() {
	vm.importCache.flushValueCache()
}

// ExtVar binds a Jsonnet external var to the given value.
func (vm *VM) ExtVar(key string, val string) {
	vm.ext[key] = vmExt{value: val, isCode: false}
	vm.flushValueCache()
}

// ExtCode binds a Jsonnet external code var to the given code.
func (vm *VM) ExtCode(key string, val string) {
	vm.ext[key] = vmExt{value: val, isCode: true}
	vm.flushValueCache()
}

// TLAVar binds a Jsonnet top level argument to the given value.
func (vm *VM) TLAVar(key string, val string) {
	vm.tla[key] = vmExt{value: val, isCode: false}
	// Setting a TLA does not require flushing the cache.
	// Only the results of evaluation of imported files are cached
	// and the TLAs do not affect these unlike extVars.
}

// TLACode binds a Jsonnet top level argument to the given code.
func (vm *VM) TLACode(key string, val string) {
	vm.tla[key] = vmExt{value: val, isCode: true}
	// Setting a TLA does not require flushing the cache - see above.
}

// Importer sets Importer to use during evaluation (import callback).
func (vm *VM) Importer(i Importer) {
	vm.importer = i
	vm.flushCache()
}

// NativeFunction registers a native function.
func (vm *VM) NativeFunction(f *NativeFunction) {
	vm.nativeFuncs[f.Name] = f
	vm.flushValueCache()
}

type evalKind int

const (
	evalKindRegular evalKind = iota
	evalKindMulti            = iota
	evalKindStream           = iota
)

// version is the current gojsonnet's version
const version = "v0.17.0"

// Evaluate evaluates a Jsonnet program given by an Abstract Syntax Tree
// and returns serialized JSON as string.
// TODO(sbarzowski) perhaps is should return JSON in standard Go representation
func (vm *VM) Evaluate(node ast.Node) (val string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("(CRASH) %v\n%s", r, debug.Stack())
		}
	}()
	return evaluate(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importCache, vm.StringOutput)
}

// EvaluateStream evaluates a Jsonnet program given by an Abstract Syntax Tree
// and returns an array of JSON strings.
func (vm *VM) EvaluateStream(node ast.Node) (output []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("(CRASH) %v\n%s", r, debug.Stack())
		}
	}()
	return evaluateStream(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importCache)
}

// EvaluateMulti evaluates a Jsonnet program given by an Abstract Syntax Tree
// and returns key-value pairs.
// The keys are strings and the values are JSON strigns (serialized JSON).
func (vm *VM) EvaluateMulti(node ast.Node) (output map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("(CRASH) %v\n%s", r, debug.Stack())
		}
	}()
	return evaluateMulti(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importCache, vm.StringOutput)
}

func (vm *VM) evaluateSnippet(diagnosticFileName ast.DiagnosticFileName, filename string, snippet string, kind evalKind) (output interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("(CRASH) %v\n%s", r, debug.Stack())
		}
	}()
	node, err := program.SnippetToAST(diagnosticFileName, filename, snippet)
	if err != nil {
		return "", err
	}
	switch kind {
	case evalKindRegular:
		output, err = evaluate(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importCache, vm.StringOutput)
	case evalKindMulti:
		output, err = evaluateMulti(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importCache, vm.StringOutput)
	case evalKindStream:
		output, err = evaluateStream(node, vm.ext, vm.tla, vm.nativeFuncs, vm.MaxStack, vm.importCache)
	}
	if err != nil {
		return "", err
	}
	return output, nil
}

func getAbsPath(path string) (string, error) {
	var absPath string
	if filepath.IsAbs(path) {
		absPath = path
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return "", nil
		}
		absPath = strings.Join([]string{wd, path}, string(filepath.Separator))
	}
	cleanedAbsPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	return cleanedAbsPath, nil
}

func (vm *VM) findDependencies(filePath string, node *ast.Node, dependencies map[string]struct{}, stackTrace *[]traceFrame) (err error) {
	var cleanedAbsPath string
	switch i := (*node).(type) {
	case *ast.Import:
		node, foundAt, err := vm.ImportAST(filePath, i.File.Value)
		if err != nil {
			*stackTrace = append([]traceFrame{{Loc: *i.Loc()}}, *stackTrace...)
			return err
		}
		cleanedAbsPath = foundAt
		if _, isFileImporter := vm.importer.(*FileImporter); isFileImporter {
			cleanedAbsPath, err = getAbsPath(foundAt)
			if err != nil {
				*stackTrace = append([]traceFrame{{Loc: *i.Loc()}}, *stackTrace...)
				return err
			}
		}
		// Check that we haven't already parsed the imported file.
		if _, alreadyParsed := dependencies[cleanedAbsPath]; alreadyParsed {
			return nil
		}
		dependencies[cleanedAbsPath] = struct{}{}
		err = vm.findDependencies(foundAt, &node, dependencies, stackTrace)
		if err != nil {
			*stackTrace = append([]traceFrame{{Loc: *i.Loc()}}, *stackTrace...)
			return err
		}
	case *ast.ImportStr:
		foundAt, err := vm.ResolveImport(filePath, i.File.Value)
		if err != nil {
			*stackTrace = append([]traceFrame{{Loc: *i.Loc()}}, *stackTrace...)
			return err
		}
		cleanedAbsPath = foundAt
		if _, isFileImporter := vm.importer.(*FileImporter); isFileImporter {
			cleanedAbsPath, err = getAbsPath(foundAt)
			if err != nil {
				*stackTrace = append([]traceFrame{{Loc: *i.Loc()}}, *stackTrace...)
				return err
			}
		}
		dependencies[cleanedAbsPath] = struct{}{}
	default:
		for _, node := range parser.Children(i) {
			err = vm.findDependencies(filePath, &node, dependencies, stackTrace)
			if err != nil {
				return err
			}
		}
	}
	return
}

// EvaluateSnippet evaluates a string containing Jsonnet code, return a JSON
// string.
//
// The filename parameter is used for resolving relative imports and for errors messages.
//
// Deprecated: Use EvaluateFile or EvaluateAnonymousSnippet instead.
func (vm *VM) EvaluateSnippet(filename string, snippet string) (json string, formattedErr error) {
	output, err := vm.evaluateSnippet(ast.DiagnosticFileName(filename), filename, snippet, evalKindRegular)
	if err != nil {
		return "", errors.New(vm.ErrorFormatter.Format(err))
	}
	json = output.(string)
	return
}

// EvaluateSnippetStream evaluates a string containing Jsonnet code to an array.
// The array is returned as an array of JSON strings.
//
// The filename parameter is used for resolving relative imports and for errors messages.
//
// Deprecated: Use EvaluateFileStream or EvaluateAnonymousSnippetStream instead.
func (vm *VM) EvaluateSnippetStream(filename string, snippet string) (docs []string, formattedErr error) {
	output, err := vm.evaluateSnippet(ast.DiagnosticFileName(filename), filename, snippet, evalKindStream)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	docs = output.([]string)
	return
}

// EvaluateSnippetMulti evaluates a string containing Jsonnet code to key-value
// pairs. The keys are field name strings and the values are JSON strings.
//
// The filename parameter is used for resolving relative imports and for errors messages.
//
// Deprecated: Use EvaluateFileMulti or EvaluateAnonymousSnippetMulti instead.
func (vm *VM) EvaluateSnippetMulti(filename string, snippet string) (files map[string]string, formattedErr error) {
	output, err := vm.evaluateSnippet(ast.DiagnosticFileName(filename), filename, snippet, evalKindMulti)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	files = output.(map[string]string)
	return
}

// EvaluateAnonymousSnippet evaluates a string containing Jsonnet code, return a JSON
// string.
//
// The filename parameter is only used for error messages.
func (vm *VM) EvaluateAnonymousSnippet(filename string, snippet string) (json string, formattedErr error) {
	output, err := vm.evaluateSnippet(ast.DiagnosticFileName(filename), "", snippet, evalKindRegular)
	if err != nil {
		return "", errors.New(vm.ErrorFormatter.Format(err))
	}
	json = output.(string)
	return
}

// EvaluateAnonymousSnippetStream evaluates a string containing Jsonnet code to an array.
// The array is returned as an array of JSON strings.
//
// The filename parameter is only used for error messages.
func (vm *VM) EvaluateAnonymousSnippetStream(filename string, snippet string) (docs []string, formattedErr error) {
	output, err := vm.evaluateSnippet(ast.DiagnosticFileName(filename), "", snippet, evalKindStream)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	docs = output.([]string)
	return
}

// EvaluateAnonymousSnippetMulti evaluates a string containing Jsonnet code to key-value
// pairs. The keys are field name strings and the values are JSON strings.
//
// The filename parameter is only used for error messages.
func (vm *VM) EvaluateAnonymousSnippetMulti(filename string, snippet string) (files map[string]string, formattedErr error) {
	output, err := vm.evaluateSnippet(ast.DiagnosticFileName(filename), "", snippet, evalKindMulti)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	files = output.(map[string]string)
	return
}

// EvaluateFile evaluates Jsonnet code in a file and returns a JSON
// string.
//
// The importer is used to fetch the contents of the file.
func (vm *VM) EvaluateFile(filename string) (json string, formattedErr error) {
	node, _, err := vm.ImportAST("", filename)
	if err != nil {
		return "", errors.New(vm.ErrorFormatter.Format(err))
	}
	output, err := vm.Evaluate(node)
	if err != nil {
		return "", errors.New(vm.ErrorFormatter.Format(err))
	}
	return output, nil
}

// EvaluateFileStream evaluates Jsonnet code in a file to an array.
// The array is returned as an array of JSON strings.
//
// The importer is used to fetch the contents of the file.
func (vm *VM) EvaluateFileStream(filename string) (docs []string, formattedErr error) {
	node, _, err := vm.ImportAST("", filename)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	output, err := vm.EvaluateStream(node)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	return output, nil
}

// EvaluateFileMulti evaluates Jsonnet code in a file to key-value
// pairs. The keys are field name strings and the values are JSON strings.
//
// The importer is used to fetch the contents of the file.
func (vm *VM) EvaluateFileMulti(filename string) (files map[string]string, formattedErr error) {
	node, _, err := vm.ImportAST("", filename)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	output, err := vm.EvaluateMulti(node)
	if err != nil {
		return nil, errors.New(vm.ErrorFormatter.Format(err))
	}
	return output, nil
}

// FindDependencies returns a sorted array of unique transitive dependencies (via import or importstr)
// from all the given `importedPaths` which are themselves excluded from the returned array.
// The `importedPaths` are parsed as if they were imported from a Jsonnet file located at `importedFrom`.
func (vm *VM) FindDependencies(importedFrom string, importedPaths []string) ([]string, error) {
	var nodes []*ast.Node
	var stackTrace []traceFrame
	filePaths := make([]string, len(importedPaths))
	depsToExclude := make([]string, len(importedPaths))
	deps := make(map[string]struct{})

	for i, filePath := range importedPaths {
		node, foundAt, err := vm.ImportAST(importedFrom, filePath)
		if err != nil {
			return nil, err
		}
		cleanedAbsPath := foundAt
		if _, isFileImporter := vm.importer.(*FileImporter); isFileImporter {
			cleanedAbsPath, err = getAbsPath(foundAt)
			if err != nil {
				return nil, err
			}
		}
		nodes = append(nodes, &node)
		filePaths[i] = foundAt

		// Add `importedPaths` to the dependencies so that they are not parsed again.
		// Will be removed before returning.
		deps[cleanedAbsPath] = struct{}{}
		depsToExclude[i] = cleanedAbsPath
	}

	for i, filePath := range filePaths {
		err := vm.findDependencies(filePath, nodes[i], deps, &stackTrace)
		if err != nil {
			err = makeRuntimeError(err.Error(), stackTrace)
			return nil, errors.New(vm.ErrorFormatter.Format(err))
		}
	}

	// Exclude `importedPaths` from the dependencies.
	for _, dep := range depsToExclude {
		delete(deps, dep)
	}

	dependencies, i := make([]string, len(deps)), 0
	for key := range deps {
		dependencies[i] = key
		i++
	}
	sort.Strings(dependencies)

	return dependencies, nil
}

// ResolveImport finds the actual path where the imported file can be found.
// It will cache the contents of the file immediately as well, to avoid the possibility of the file
// disappearing after being checked.
func (vm *VM) ResolveImport(importedFrom, importedPath string) (foundAt string, err error) {
	_, foundAt, err = vm.importCache.importData(importedFrom, importedPath)
	return
}

// ImportData fetches the data just as if it was imported from a Jsonnet file located at `importedFrom`.
// It shares the cache with the actual evaluation.
func (vm *VM) ImportData(importedFrom, importedPath string) (contents string, foundAt string, err error) {
	c, foundAt, err := vm.importCache.importData(importedFrom, importedPath)
	if err != nil {
		return "", foundAt, err
	}
	return c.String(), foundAt, err
}

// ImportAST fetches the Jsonnet AST just as if it was imported from a Jsonnet file located at `importedFrom`.
// It shares the cache with the actual evaluation.
func (vm *VM) ImportAST(importedFrom, importedPath string) (contents ast.Node, foundAt string, err error) {
	return vm.importCache.importAST(importedFrom, importedPath)
}

// SnippetToAST parses a snippet and returns the resulting AST.
func SnippetToAST(filename string, snippet string) (ast.Node, error) {
	return program.SnippetToAST(ast.DiagnosticFileName(filename), filename, snippet)
}

// Version returns the Jsonnet version number.
func Version() string {
	return version
}
